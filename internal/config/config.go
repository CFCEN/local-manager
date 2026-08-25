package config

import (
	"os"
	"path/filepath"
)

type AppConfig struct {
	Port          string
	DBPath        string
	WorkspaceRoot string
}

func Load() *AppConfig {
	root := envOrDefault("WORKSPACE_ROOT", defaultWorkspaceRoot())
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}
	return &AppConfig{
		Port:          envOrDefault("PORT", "9527"),
		DBPath:        envOrDefault("DB_PATH", "data/workspace.db"),
		WorkspaceRoot: root,
	}
}

func defaultWorkspaceRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "workspace"
	}
	return filepath.Join(home, ".local-dev-workspace", "workspace")
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
