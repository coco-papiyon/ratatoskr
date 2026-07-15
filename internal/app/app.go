package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// App contains the desktop-facing services bound to the frontend.
type App struct {
	ctx                  context.Context
	viewerConfig         ViewerConfig
	structuredTableRules []StructuredTableRule
}

func NewApp() *App {
	return &App{viewerConfig: DefaultViewerConfig(), structuredTableRules: DefaultStructuredTableRules()}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	if config, err := loadViewerConfig(); err == nil {
		a.viewerConfig = config
	}
	if rules, err := loadStructuredTableRules(); err == nil {
		a.structuredTableRules = rules
	}
}

func (a *App) AppInfo() string {
	return "Ratatoskr desktop backend is ready"
}

func (a *App) CurrentWorkingDirectory() (string, error) {
	path, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("カレントディレクトリを取得できません: %w", err)
	}
	return filepath.Abs(path)
}

func (a *App) ParentLocalDirectory(path string) (string, error) {
	path = filepath.Clean(path)
	parent := filepath.Dir(path)
	if parent == path {
		return "", fmt.Errorf("これ以上上のディレクトリへは移動できません")
	}
	return parent, nil
}
