package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Env         string            `yaml:"env"` // "dev" or "prod"
	Server      ServerConfig      `yaml:"server"`
	Database    DatabaseConfig    `yaml:"database"`
	Log         LogConfig         `yaml:"log"`
	Auth        AuthConfig        `yaml:"auth"`
	SSG         SSGConfig         `yaml:"ssg"`
	Credentials CredentialsConfig `yaml:"credentials"`
	LLM         LLMConfig         `yaml:"llm"`
}

func (c *Config) IsDev() bool {
	return c.Env == "dev"
}

type ServerConfig struct {
	Addr string `yaml:"addr"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type LogConfig struct {
	Level string `yaml:"level"`
}

type AuthConfig struct {
	SessionSecret string `yaml:"session_secret"`
	SessionTTL    string `yaml:"session_ttl"`
}

type SSGConfig struct {
	SitesBasePath string `yaml:"sites_base_path"`
	PreviewAddr   string `yaml:"preview_addr"`
}

type CredentialsConfig struct {
	Path string `yaml:"path"`
}

type LLMConfig struct {
	Provider    string  `yaml:"provider"`    // "openai" (default)
	APIKey      string  `yaml:"api_key"`
	Model       string  `yaml:"model"`       // default: "gpt-4o"
	Temperature float64 `yaml:"temperature"` // default: 0.3
}

func Load() *Config {
	// Determine environment first
	env := os.Getenv("CLIO_ENV")
	if env == "" {
		env = "dev" // Default to dev for safety
	}

	// Set paths based on environment
	var dbPath, sitesPath string
	if env == "dev" {
		// Dev: use local project directories
		dbPath = "_workspace/db/clio.db"
		sitesPath = "_workspace/sites"
	} else {
		// Prod: use system directories
		homeDir, _ := os.UserHomeDir()
		dbPath = filepath.Join(homeDir, ".clio", "clio.db")
		sitesPath = filepath.Join(homeDir, "Documents", "Clio", "sites")
	}

	cfg := &Config{
		Env:      env,
		Server:   ServerConfig{Addr: ":8080"},
		Database: DatabaseConfig{Path: dbPath},
		Log:      LogConfig{Level: "info"},
		Auth:     AuthConfig{SessionTTL: "720h"}, // 30 days
		SSG:      SSGConfig{SitesBasePath: sitesPath, PreviewAddr: ":3000"},
		LLM:      LLMConfig{Provider: "openai", Model: "gpt-4o", Temperature: 0.3},
	}

	data, err := os.ReadFile("config.yaml")
	if err == nil {
		yaml.Unmarshal(data, cfg)
	}

	// Environment overrides (highest priority)
	if v := os.Getenv("CLIO_SERVER_ADDR"); v != "" {
		cfg.Server.Addr = v
	}
	if v := os.Getenv("CLIO_DATABASE_PATH"); v != "" {
		cfg.Database.Path = v
	}
	if v := os.Getenv("CLIO_LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}
	if v := os.Getenv("CLIO_AUTH_SESSION_SECRET"); v != "" {
		cfg.Auth.SessionSecret = v
	}
	if v := os.Getenv("CLIO_CREDENTIALS_PATH"); v != "" {
		cfg.Credentials.Path = v
	}
	if v := os.Getenv("CLIO_SSG_SITES_PATH"); v != "" {
		cfg.SSG.SitesBasePath = v
	}
	if v := os.Getenv("CLIO_SSG_PREVIEW_ADDR"); v != "" {
		cfg.SSG.PreviewAddr = v
	}
	if v := os.Getenv("OPENAI_API_KEY"); v != "" && cfg.LLM.APIKey == "" {
		cfg.LLM.APIKey = v
	}
	if v := os.Getenv("CLIO_LLM_API_KEY"); v != "" {
		cfg.LLM.APIKey = v
	}
	if v := os.Getenv("CLIO_LLM_MODEL"); v != "" {
		cfg.LLM.Model = v
	}

	return cfg
}
