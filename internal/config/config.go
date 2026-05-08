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
	Upload    UploadConfig    `yaml:"upload"`
}

type HTTPConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

type MaiyatianConfig struct {
	Cookie    string        `yaml:"cookie"`
	UserAgent string        `yaml:"user_agent"`
	Timeout   time.Duration `yaml:"timeout"`
}

type UploadConfig struct {
	BaseURL string        `yaml:"base_url"`
	APIKey  string        `yaml:"api_key"`
	Timeout time.Duration `yaml:"timeout"`
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
			UserAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36",
			Timeout:   15 * time.Second,
		},
		Upload: UploadConfig{
			BaseURL: "http://127.0.0.1:8850",
			APIKey:  "ms_fece8dc13ae84ef8b39ff827c4bfe09cebcc422aa862ab1e",
			Timeout: 15 * time.Second,
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

	if value := strings.TrimSpace(src.Maiyatian.Cookie); value != "" {
		dst.Maiyatian.Cookie = value
	}
	if value := strings.TrimSpace(src.Maiyatian.UserAgent); value != "" {
		dst.Maiyatian.UserAgent = value
	}
	if src.Maiyatian.Timeout > 0 {
		dst.Maiyatian.Timeout = src.Maiyatian.Timeout
	}

	if value := strings.TrimSpace(src.Upload.BaseURL); value != "" {
		dst.Upload.BaseURL = strings.TrimRight(value, "/")
	}
	if value := strings.TrimSpace(src.Upload.APIKey); value != "" {
		dst.Upload.APIKey = value
	}
	if src.Upload.Timeout > 0 {
		dst.Upload.Timeout = src.Upload.Timeout
	}
}
