package app

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxArchiveSize = 256 * 1024 * 1024
const maxPreviewSize = 4 * 1024 * 1024

type archiveMember struct {
	name       string
	isDir      bool
	size       int64
	modifiedAt string
}

func (a *App) ListLocalArchive(archivePath, prefix string) ([]LocalEntry, error) {
	contents, err := readLocalArchive(archivePath)
	if err != nil {
		return nil, err
	}
	return listArchiveEntries(archivePath, prefix, contents)
}

func (a *App) ReadLocalArchivePreview(archivePath, entryPath, charset string) (S3Preview, error) {
	contents, err := readLocalArchive(archivePath)
	if err != nil {
		return S3Preview{}, err
	}
	return readArchivePreview(archivePath, entryPath, charset, contents)
}

func readLocalArchive(archivePath string) ([]byte, error) {
	archivePath = filepath.Clean(archivePath)
	info, err := os.Stat(archivePath)
	if err != nil {
		return nil, fmt.Errorf("圧縮ファイルを確認できません: %w", err)
	}
	if info.Size() > maxArchiveSize {
		return nil, fmt.Errorf("圧縮ファイルが大きすぎます（最大 256 MB）")
	}
	contents, err := os.ReadFile(archivePath)
	if err != nil {
		return nil, fmt.Errorf("圧縮ファイルを読み込めません: %w", err)
	}
	return contents, nil
}

func listArchiveEntries(archivePath, prefix string, contents []byte) ([]LocalEntry, error) {
	members, err := archiveMembers(archivePath, contents)
	if err != nil {
		return nil, err
	}
	prefix = normalizeArchivePrefix(prefix)
	children := make(map[string]LocalEntry)
	for _, member := range members {
		name := normalizeArchiveEntry(member.name)
		if name == "" || !strings.HasPrefix(name, prefix) {
			continue
		}
		relative := strings.TrimPrefix(name, prefix)
		if relative == "" {
			continue
		}
		parts := strings.SplitN(relative, "/", 2)
		childName := parts[0]
		isFolder := len(parts) > 1 || member.isDir
		childEntry := prefix + childName
		kind := "file"
		if isFolder {
			kind = "folder"
			childEntry += "/"
		}
		if _, exists := children[childEntry]; exists {
			continue
		}
		entry := LocalEntry{
			ID: archivePath + "!/" + childEntry, Name: childName, Kind: kind,
			Path: archivePath + "!/" + childEntry, ArchivePath: archivePath, ArchiveEntry: childEntry,
		}
		if !isFolder {
			entry.Size = member.size
			entry.ModifiedAt = member.modifiedAt
		}
		children[childEntry] = entry
	}
	entries := make([]LocalEntry, 0, len(children))
	for _, entry := range children {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind == "folder"
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	return entries, nil
}

func archiveMembers(archivePath string, contents []byte) ([]archiveMember, error) {
	format := archiveFormat(archivePath)
	switch format {
	case "zip":
		reader, err := zip.NewReader(bytes.NewReader(contents), int64(len(contents)))
		if err != nil {
			return nil, fmt.Errorf("ZIPを開けません: %w", err)
		}
		members := make([]archiveMember, 0, len(reader.File))
		for _, file := range reader.File {
			members = append(members, archiveMember{name: file.Name, isDir: file.FileInfo().IsDir(), size: int64(file.UncompressedSize64), modifiedAt: formatModifiedAt(file.Modified)})
		}
		return members, nil
	case "tar", "targz":
		reader, closeReader, err := newTarReader(format, contents)
		if err != nil {
			return nil, err
		}
		defer closeReader()
		members := make([]archiveMember, 0)
		for {
			header, nextErr := reader.Next()
			if nextErr == io.EOF {
				break
			}
			if nextErr != nil {
				return nil, fmt.Errorf("TARを読み込めません: %w", nextErr)
			}
			members = append(members, archiveMember{name: header.Name, isDir: header.FileInfo().IsDir(), size: header.Size, modifiedAt: formatModifiedAt(header.ModTime)})
		}
		return members, nil
	default:
		return nil, fmt.Errorf("対応していない圧縮形式です: %s", archivePath)
	}
}

func readArchivePreview(archivePath, entryPath, charset string, contents []byte) (S3Preview, error) {
	entryContents, err := readArchiveEntry(archivePath, entryPath, contents)
	if err != nil {
		return S3Preview{}, err
	}
	contentType := mime.TypeByExtension(filepath.Ext(entryPath))
	if strings.HasPrefix(contentType, "image/") {
		return S3Preview{DataURL: "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(entryContents)}, nil
	}
	text, err := decodeText(entryContents, charset)
	if err != nil {
		return S3Preview{}, err
	}
	return S3Preview{Content: text}, nil
}

func readArchiveEntry(archivePath, entryPath string, contents []byte) ([]byte, error) {
	target := normalizeArchiveEntry(entryPath)
	format := archiveFormat(archivePath)
	switch format {
	case "zip":
		reader, err := zip.NewReader(bytes.NewReader(contents), int64(len(contents)))
		if err != nil {
			return nil, fmt.Errorf("ZIPを開けません: %w", err)
		}
		for _, file := range reader.File {
			if normalizeArchiveEntry(file.Name) != target || file.FileInfo().IsDir() {
				continue
			}
			if file.UncompressedSize64 > maxPreviewSize {
				return nil, fmt.Errorf("プレビューできるサイズを超えています（最大 4 MB）")
			}
			opened, openErr := file.Open()
			if openErr != nil {
				return nil, fmt.Errorf("圧縮ファイル内の項目を開けません: %w", openErr)
			}
			defer opened.Close()
			return readBoundedPreview(opened)
		}
	case "tar", "targz":
		reader, closeReader, err := newTarReader(format, contents)
		if err != nil {
			return nil, err
		}
		defer closeReader()
		for {
			header, nextErr := reader.Next()
			if nextErr == io.EOF {
				break
			}
			if nextErr != nil {
				return nil, fmt.Errorf("TARを読み込めません: %w", nextErr)
			}
			if normalizeArchiveEntry(header.Name) == target && !header.FileInfo().IsDir() {
				if header.Size > maxPreviewSize {
					return nil, fmt.Errorf("プレビューできるサイズを超えています（最大 4 MB）")
				}
				return readBoundedPreview(reader)
			}
		}
	}
	return nil, fmt.Errorf("圧縮ファイル内に項目が見つかりません: %s", entryPath)
}

func newTarReader(format string, contents []byte) (*tar.Reader, func(), error) {
	baseReader := bytes.NewReader(contents)
	if format == "targz" {
		gzipReader, err := gzip.NewReader(baseReader)
		if err != nil {
			return nil, func() {}, fmt.Errorf("GZIPを開けません: %w", err)
		}
		return tar.NewReader(gzipReader), func() { _ = gzipReader.Close() }, nil
	}
	return tar.NewReader(baseReader), func() {}, nil
}

func readBoundedPreview(reader io.Reader) ([]byte, error) {
	contents, err := io.ReadAll(io.LimitReader(reader, maxPreviewSize+1))
	if err != nil {
		return nil, fmt.Errorf("圧縮ファイル内の項目を読み込めません: %w", err)
	}
	if len(contents) > maxPreviewSize {
		return nil, fmt.Errorf("プレビューできるサイズを超えています（最大 4 MB）")
	}
	return contents, nil
}

func archiveFormat(path string) string {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return "zip"
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return "targz"
	case strings.HasSuffix(lower, ".tar"):
		return "tar"
	default:
		return ""
	}
}

func normalizeArchivePrefix(prefix string) string {
	prefix = normalizeArchiveEntry(prefix)
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return prefix
}

func normalizeArchiveEntry(entry string) string {
	entry = strings.ReplaceAll(entry, "\\", "/")
	entry = strings.TrimPrefix(entry, "./")
	return strings.Trim(entry, "/")
}
