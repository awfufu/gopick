package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	HTTP      HTTPConfig      `yaml:"http"`
	Maiyatian MaiyatianConfig `yaml:"maiyatian"`
}

type HTTPConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

type MaiyatianConfig struct {
	BaseURL    string        `yaml:"base_url"`
	APIKey     string        `yaml:"api_key"`
	APISecret  string        `yaml:"api_secret"`
	Cookie     string        `yaml:"cookie"`
	UserAgent  string        `yaml:"user_agent"`
	Timeout    time.Duration `yaml:"timeout"`
	WebhookKey string        `yaml:"webhook_key"`
}

func Load(path string) (Config, error) {
	cfg := Config{
		HTTP: HTTPConfig{
			Host:            "127.0.0.1",
			Port:            22800,
			ReadTimeout:     15 * time.Second,
			WriteTimeout:    15 * time.Second,
			ShutdownTimeout: 10 * time.Second,
		},
		Maiyatian: MaiyatianConfig{
			BaseURL:   "https://saas.maiyatian.com",
			UserAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36",
			Timeout:   15 * time.Second,
		},
	}

	if strings.TrimSpace(path) == "" {
		return Config{}, fmt.Errorf("config file path is required")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config file %q: %w", path, err)
	}

	fileCfg := Config{}
	if err := yaml.Unmarshal(content, &fileCfg); err != nil {
		return Config{}, fmt.Errorf("parse config file %q: %w", path, err)
	}

	mergeConfig(&cfg, fileCfg)
	return cfg, nil
}

func mergeConfig(dst *Config, src Config) {
	if value := strings.TrimSpace(src.HTTP.Host); value != "" {
		dst.HTTP.Host = value
	}
	if src.HTTP.Port > 0 {
		dst.HTTP.Port = src.HTTP.Port
	}
	if src.HTTP.ReadTimeout > 0 {
		dst.HTTP.ReadTimeout = src.HTTP.ReadTimeout
	}
	if src.HTTP.WriteTimeout > 0 {
		dst.HTTP.WriteTimeout = src.HTTP.WriteTimeout
	}
	if src.HTTP.ShutdownTimeout > 0 {
		dst.HTTP.ShutdownTimeout = src.HTTP.ShutdownTimeout
	}

	if value := strings.TrimSpace(src.Maiyatian.BaseURL); value != "" {
		dst.Maiyatian.BaseURL = strings.TrimRight(value, "/")
	}
	if value := strings.TrimSpace(src.Maiyatian.APIKey); value != "" {
		dst.Maiyatian.APIKey = value
	}
	if value := strings.TrimSpace(src.Maiyatian.APISecret); value != "" {
		dst.Maiyatian.APISecret = value
	}
	if value := strings.TrimSpace(src.Maiyatian.Cookie); value != "" {
		dst.Maiyatian.Cookie = value
	}
	if value := strings.TrimSpace(src.Maiyatian.UserAgent); value != "" {
		dst.Maiyatian.UserAgent = value
	}
	if src.Maiyatian.Timeout > 0 {
		dst.Maiyatian.Timeout = src.Maiyatian.Timeout
	}
	if value := strings.TrimSpace(src.Maiyatian.WebhookKey); value != "" {
		dst.Maiyatian.WebhookKey = value
	}
}
