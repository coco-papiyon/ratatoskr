package app

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func (a *App) ListAWSProfiles() []string {
	profiles := map[string]bool{"default": true}
	home, err := os.UserHomeDir()
	if err != nil {
		return []string{"default"}
	}
	for _, path := range []string{filepath.Join(home, ".aws", "config"), filepath.Join(home, ".aws", "credentials")} {
		collectAWSProfiles(path, profiles)
	}
	result := make([]string, 0, len(profiles))
	for profile := range profiles {
		result = append(result, profile)
	}
	sort.Strings(result)
	return result
}

func (a *App) ListS3Buckets(profile, region string) ([]LocalEntry, error) {
	client, err := a.s3Client(profile, region)
	if err != nil {
		return nil, err
	}
	response, err := client.ListBuckets(a.ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("S3 バケット一覧を取得できません: %w", err)
	}
	entries := make([]LocalEntry, 0, len(response.Buckets))
	for _, bucket := range response.Buckets {
		name := aws.ToString(bucket.Name)
		entries = append(entries, LocalEntry{ID: "s3://" + name, Name: name, Kind: "folder", Path: "s3://" + name, ModifiedAt: formatModifiedAt(aws.ToTime(bucket.CreationDate))})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return entries, nil
}

func (a *App) ListS3Directory(profile, region, bucket, prefix string) ([]LocalEntry, error) {
	if bucket == "" {
		return nil, fmt.Errorf("S3 バケットが指定されていません")
	}
	client, err := a.s3Client(profile, region)
	if err != nil {
		return nil, err
	}
	prefix = strings.TrimPrefix(prefix, "/")
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	response, err := client.ListObjectsV2(a.ctx, &s3.ListObjectsV2Input{Bucket: aws.String(bucket), Prefix: aws.String(prefix), Delimiter: aws.String("/")})
	if err != nil {
		return nil, fmt.Errorf("S3 の一覧を取得できません: %w", err)
	}
	entries := make([]LocalEntry, 0, len(response.CommonPrefixes)+len(response.Contents))
	for _, commonPrefix := range response.CommonPrefixes {
		key := aws.ToString(commonPrefix.Prefix)
		name := strings.TrimSuffix(strings.TrimPrefix(key, prefix), "/")
		entries = append(entries, LocalEntry{ID: "s3://" + bucket + "/" + key, Name: name, Kind: "folder", Path: "s3://" + bucket + "/" + key})
	}
	for _, object := range response.Contents {
		key := aws.ToString(object.Key)
		if key == prefix {
			continue
		}
		entries = append(entries, LocalEntry{ID: "s3://" + bucket + "/" + key, Name: strings.TrimPrefix(key, prefix), Kind: "file", Path: "s3://" + bucket + "/" + key, ModifiedAt: formatModifiedAt(aws.ToTime(object.LastModified)), Size: aws.ToInt64(object.Size)})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind == "folder"
		}
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}

func (a *App) ReadS3Preview(profile, region, bucket, key, charset string) (S3Preview, error) {
	client, err := a.s3Client(profile, region)
	if err != nil {
		return S3Preview{}, err
	}
	response, err := client.GetObject(a.ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	if err != nil {
		return S3Preview{}, fmt.Errorf("S3 オブジェクトを取得できません: %w", err)
	}
	defer response.Body.Close()
	contents, err := io.ReadAll(io.LimitReader(response.Body, maxPreviewSize+1))
	if err != nil {
		return S3Preview{}, fmt.Errorf("S3 オブジェクトを読み込めません: %w", err)
	}
	if len(contents) > maxPreviewSize {
		return S3Preview{}, fmt.Errorf("プレビューできるサイズを超えています（最大 4 MB）")
	}
	contentType := aws.ToString(response.ContentType)
	if strings.HasPrefix(contentType, "image/") {
		return S3Preview{DataURL: "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(contents)}, nil
	}
	text, err := decodeText(contents, charset)
	if err != nil {
		return S3Preview{}, err
	}
	return S3Preview{Content: text}, nil
}

func (a *App) ListS3Archive(profile, region, bucket, key, prefix string) ([]LocalEntry, error) {
	contents, err := a.readS3Archive(profile, region, bucket, key)
	if err != nil {
		return nil, err
	}
	return listArchiveEntries("s3://"+bucket+"/"+key, prefix, contents)
}

func (a *App) ReadS3ArchivePreview(profile, region, bucket, key, entryPath, charset string) (S3Preview, error) {
	contents, err := a.readS3Archive(profile, region, bucket, key)
	if err != nil {
		return S3Preview{}, err
	}
	return readArchivePreview(key, entryPath, charset, contents)
}

func (a *App) readS3Archive(profile, region, bucket, key string) ([]byte, error) {
	client, err := a.s3Client(profile, region)
	if err != nil {
		return nil, err
	}
	response, err := client.GetObject(a.ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	if err != nil {
		return nil, fmt.Errorf("S3の圧縮ファイルを取得できません: %w", err)
	}
	defer response.Body.Close()
	if aws.ToInt64(response.ContentLength) > maxArchiveSize {
		return nil, fmt.Errorf("圧縮ファイルが大きすぎます（最大 256 MB）")
	}
	contents, err := io.ReadAll(io.LimitReader(response.Body, maxArchiveSize+1))
	if err != nil {
		return nil, fmt.Errorf("S3の圧縮ファイルを読み込めません: %w", err)
	}
	if len(contents) > maxArchiveSize {
		return nil, fmt.Errorf("圧縮ファイルが大きすぎます（最大 256 MB）")
	}
	return contents, nil
}

func (a *App) s3Client(profile, region string) (*s3.Client, error) {
	options := []func(*awsconfig.LoadOptions) error{}
	if profile != "" {
		options = append(options, awsconfig.WithSharedConfigProfile(profile))
	}
	if region != "" {
		options = append(options, awsconfig.WithRegion(region))
	}
	httpClient, err := a.httpClient()
	if err != nil {
		return nil, err
	}
	options = append(options, awsconfig.WithHTTPClient(httpClient))
	config, err := awsconfig.LoadDefaultConfig(a.ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("AWS 設定を読み込めません: %w", err)
	}
	return s3.NewFromConfig(config), nil
}

func (a *App) httpClient() (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if a.viewerConfig.Proxy != "" {
		proxyURL, err := url.Parse(a.viewerConfig.Proxy)
		if err != nil || proxyURL.Scheme == "" || proxyURL.Host == "" {
			return nil, fmt.Errorf("Proxy URLが不正です: %s", a.viewerConfig.Proxy)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	if a.viewerConfig.Certificate != "" {
		certificate, err := os.ReadFile(a.viewerConfig.Certificate)
		if err != nil {
			return nil, fmt.Errorf("CA証明書を読み込めません: %w", err)
		}
		roots, err := x509.SystemCertPool()
		if err != nil || roots == nil {
			roots = x509.NewCertPool()
		}
		if !roots.AppendCertsFromPEM(certificate) {
			return nil, fmt.Errorf("CA証明書の形式が不正です: %s", a.viewerConfig.Certificate)
		}
		transport.TLSClientConfig = &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}
	}
	return &http.Client{Transport: transport}, nil
}

func collectAWSProfiles(path string, profiles map[string]bool) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "[") || !strings.HasSuffix(line, "]") {
			continue
		}
		name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
		name = strings.TrimPrefix(name, "profile ")
		if name != "" {
			profiles[name] = true
		}
	}
}
