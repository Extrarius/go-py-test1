// Package config загружает .env и config.yaml.
package config

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// Config — настройки приложения (из .env и config.yaml).
type Config struct {
	// OpenRouter
	OpenRouterAPIKey string
	OpenRouterModel  string

	// Google Drive
	CredentialsPath string
	FolderID        string
	TokenPath       string

	// Кеш и пути
	CacheDir string

	// Из config.yaml (режимы, лимиты)
	Mode           string // fast | deep
	Source         string // drive | local
	ChunkTokens    int
	LLMConcurrency int
	LocalDir       string // путь к локальной папке при source=local
}

// Load загружает .env и config.yaml; .env имеет приоритет над yaml.
func Load() (*Config, error) {
	_ = godotenv.Load() // игнорируем ошибку, если .env нет

	cfg := &Config{
		OpenRouterAPIKey: os.Getenv("OPENROUTER_API_KEY"),
		OpenRouterModel:  os.Getenv("OPENROUTER_MODEL"),
		CredentialsPath:  envOr("GOOGLE_CREDENTIALS_PATH", "credentials.json"),
		FolderID:         os.Getenv("GOOGLE_DRIVE_FOLDER_ID"),
		TokenPath:        envOr("GOOGLE_TOKEN_PATH", "token.json"),
		CacheDir:         envOr("CACHE_DIR", ".cache"),
		Mode:             "fast",
		Source:           "drive",
		ChunkTokens:      4000,
		LLMConcurrency:   1,
	}
	cfg.CacheDir, _ = filepath.Abs(cfg.CacheDir)

	y, err := loadYAML("config.yaml")
	if err != nil {
		return nil, err
	}
	if y != nil {
		if y.Mode != "" {
			cfg.Mode = y.Mode
		}
		if y.Source != "" {
			cfg.Source = y.Source
		}
		if y.ChunkTokens > 0 {
			cfg.ChunkTokens = y.ChunkTokens
		}
		if y.LLMConcurrency > 0 {
			cfg.LLMConcurrency = y.LLMConcurrency
		}
		if y.Model != "" && cfg.OpenRouterModel == "" {
			cfg.OpenRouterModel = y.Model
		}
		if y.CacheDir != "" && os.Getenv("CACHE_DIR") == "" {
			cfg.CacheDir, _ = filepath.Abs(y.CacheDir)
		}
		if y.CredentialsPath != "" && os.Getenv("GOOGLE_CREDENTIALS_PATH") == "" {
			cfg.CredentialsPath = y.CredentialsPath
		}
		if y.LocalDir != "" {
			cfg.LocalDir = y.LocalDir
		}
	}
	if cfg.LocalDir == "" && cfg.Source == "local" {
		cfg.LocalDir = cfg.CacheDir
	}
	return cfg, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
