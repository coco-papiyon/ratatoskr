package app

import (
	"encoding/base64"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) SelectLocalDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "ローカルフォルダを開く"})
}

func (a *App) ListLocalDirectory(path string) ([]LocalEntry, error) {
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("フォルダを確認できません: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("フォルダではありません: %s", path)
	}
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("フォルダを読み込めません: %w", err)
	}
	entries := make([]LocalEntry, 0, len(dirEntries))
	for _, entry := range dirEntries {
		entryPath := filepath.Join(path, entry.Name())
		kind := "file"
		if entry.IsDir() {
			kind = "folder"
		}
		item := LocalEntry{ID: entryPath, Name: entry.Name(), Kind: kind, Path: entryPath}
		if metadata, metadataErr := entry.Info(); metadataErr == nil {
			item.ModifiedAt = formatModifiedAt(metadata.ModTime())
			if !metadata.IsDir() {
				item.Size = metadata.Size()
			}
		}
		entries = append(entries, item)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind == "folder"
		}
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}

func (a *App) ReadLocalTextFile(path string) (string, error) {
	contents, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("ファイルを読み込めません: %w", err)
	}
	return string(contents), nil
}

func (a *App) GetLocalFileModifiedAt(path string) (int64, error) {
	info, err := os.Stat(filepath.Clean(path))
	if err != nil {
		return 0, fmt.Errorf("ファイルを確認できません: %w", err)
	}
	return info.ModTime().UnixMilli(), nil
}

func (a *App) ReadLocalPreview(path, charset string) (S3Preview, error) {
	path = filepath.Clean(path)
	contents, err := os.ReadFile(path)
	if err != nil {
		return S3Preview{}, fmt.Errorf("ファイルを読み込めません: %w", err)
	}
	contentType := mime.TypeByExtension(filepath.Ext(path))
	if strings.HasPrefix(contentType, "image/") {
		return S3Preview{DataURL: "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(contents)}, nil
	}
	text, err := decodeText(contents, charset)
	if err != nil {
		return S3Preview{}, err
	}
	if strings.EqualFold(filepath.Ext(path), ".md") || strings.EqualFold(filepath.Ext(path), ".markdown") || strings.EqualFold(filepath.Ext(path), ".mdx") {
		text = inlineLocalMarkdownImages(text, path)
	}
	return S3Preview{Content: text}, nil
}

func inlineLocalMarkdownImages(markdown, markdownPath string) string {
	pattern := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	return pattern.ReplaceAllStringFunc(markdown, func(match string) string {
		parts := pattern.FindStringSubmatch(match)
		if len(parts) != 3 || strings.Contains(parts[2], "://") {
			return match
		}
		imagePath := filepath.Clean(filepath.Join(filepath.Dir(markdownPath), parts[2]))
		image, err := os.ReadFile(imagePath)
		if err != nil {
			return match
		}
		contentType := mime.TypeByExtension(filepath.Ext(imagePath))
		if !strings.HasPrefix(contentType, "image/") {
			return match
		}
		dataURL := "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(image)
		return "![" + parts[1] + "](" + dataURL + ")"
	})
}
