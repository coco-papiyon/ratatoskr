package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func DefaultViewerConfig() ViewerConfig {
	return ViewerConfig{Extensions: map[string][]string{
		"markdown":   {".md", ".markdown", ".mdx"},
		"text":       {".txt", ".log", ".out", ".err", ".yaml", ".yml", ".toml", ".ini", ".conf", ".properties", ".lock", ".mod", ".sum", ".md5", ".patch", ".diff", ".map", ".ts", ".tsx", ".vue", ".js", ".jsx", ".css", ".scss", ".html", ".go", ".rs", ".py", ".java", ".c", ".cpp", ".h", ".sql", ".sh", ".ps1", ".bat", ".gitignore", ".gitattributes", ".gitmodules", ".dockerignore", ".editorconfig", ".env", "dockerfile", "makefile", "procfile", "license"},
		"image":      {".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".avif"},
		"structured": {".json", ".xml", ".csv", ".yaml", ".yml"},
	}}
}

func (a *App) GetViewerConfig() ViewerConfig { return a.viewerConfig }

func (a *App) UpdateViewerConfig(config ViewerConfig) error {
	if config.Extensions == nil {
		return fmt.Errorf("拡張子設定が空です")
	}
	for category, extensions := range config.Extensions {
		if category != "markdown" && category != "text" && category != "image" && category != "structured" {
			return fmt.Errorf("不明な表示分類です: %s", category)
		}
		for index, extension := range extensions {
			extension = strings.ToLower(strings.TrimSpace(extension))
			if extension == "" {
				continue
			}
			if !strings.HasPrefix(extension, ".") && category != "text" {
				return fmt.Errorf("拡張子は . から始めてください: %s", extension)
			}
			extensions[index] = extension
		}
		config.Extensions[category] = extensions
	}
	contents, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	path, err := viewerConfigPath()
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		return fmt.Errorf("設定を保存できません: %w", err)
	}
	a.viewerConfig = config
	return nil
}

func loadViewerConfig() (ViewerConfig, error) {
	defaultConfig := DefaultViewerConfig()
	path, err := viewerConfigPath()
	if err != nil {
		return defaultConfig, err
	}
	contents, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		legacyPaths := []string{filepath.Join(filepath.Dir(path), "ratatoskr.config.json")}
		if legacyPath, legacyErr := legacyViewerConfigPath(); legacyErr == nil {
			legacyPaths = append(legacyPaths, legacyPath)
		}
		for _, legacyPath := range legacyPaths {
			if legacyContents, readErr := os.ReadFile(legacyPath); readErr == nil {
				var migrated ViewerConfig
				if json.Unmarshal(legacyContents, &migrated) == nil {
					contents, marshalErr := yaml.Marshal(migrated)
					if marshalErr != nil {
						return defaultConfig, marshalErr
					}
					if writeErr := os.WriteFile(path, contents, 0o644); writeErr != nil {
						return defaultConfig, writeErr
					}
					return migrated, nil
				}
			}
		}
		contents, err = yaml.Marshal(defaultConfig)
		if err != nil {
			return defaultConfig, err
		}
		if err := os.WriteFile(path, contents, 0o644); err != nil {
			return defaultConfig, err
		}
		return defaultConfig, nil
	}
	if err != nil {
		return defaultConfig, err
	}
	var config ViewerConfig
	if err := yaml.Unmarshal(contents, &config); err != nil {
		return defaultConfig, err
	}
	if config.Extensions == nil {
		config.Extensions = defaultConfig.Extensions
	}
	return config, nil
}

func viewerConfigPath() (string, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	configDirectory := filepath.Join(workingDirectory, "config")
	if err := os.MkdirAll(configDirectory, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(configDirectory, "ratatoskr.config.yaml"), nil
}

func legacyViewerConfigPath() (string, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(workingDirectory, "ratatoskr.config.json"), nil
}
