package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type AppConfig struct {
	Port            string   `json:"port"`
	DBPath          string   `json:"dbPath"`
	WorkspaceRoot   string   `json:"workspaceRoot"`
	AutomationToken string   `json:"automationToken"`
	Host            string   `json:"host"`
	AllowedOrigins  []string `json:"allowedOrigins"`
}

func Load() *AppConfig {
	cfg := AppConfig{
		Port:           "9527",
		DBPath:         "data/workspace.db",
		WorkspaceRoot:  defaultWorkspaceRoot(),
		Host:           "127.0.0.1",
		AllowedOrigins: []string{"http://localhost:3000", "http://127.0.0.1:3000"},
	}
	path := envOrDefault("BACKEND_CONFIG", "config.json")
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &cfg)
	}
	cfg.Port = envOrDefault("PORT", cfg.Port)
	cfg.DBPath = envOrDefault("DB_PATH", cfg.DBPath)
	cfg.WorkspaceRoot = envOrDefault("WORKSPACE_ROOT", cfg.WorkspaceRoot)
	cfg.Host = envOrDefault("HOST", cfg.Host)
	if token := os.Getenv("AUTOMATION_TOKEN"); token != "" {
		cfg.AutomationToken = token
	}
	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		cfg.AllowedOrigins = strings.FieldsFunc(origins, func(r rune) bool { return r == ',' })
	}
	if cfg.WorkspaceRoot == "" {
		cfg.WorkspaceRoot = defaultWorkspaceRoot()
	}
	if abs, err := filepath.Abs(cfg.WorkspaceRoot); err == nil {
		cfg.WorkspaceRoot = abs
	}
	return &cfg
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
