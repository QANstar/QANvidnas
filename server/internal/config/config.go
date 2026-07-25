package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server      ServerConfig      `yaml:"server"`
	DataDir     string            `yaml:"data_dir"`
	InviteCodes []InviteCodeConfig `yaml:"invite_codes"`
	ScanFolders []ScanFolderConfig `yaml:"scan_folders"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

type InviteCodeConfig struct {
	Code        string `yaml:"code"`
	Description string `yaml:"description"`
	MaxUses     int    `yaml:"max_uses"`
	ExpiresAt   string `yaml:"expires_at"`
}

type ScanFolderConfig struct {
	Path string `yaml:"path"`
}

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: "3000",
		},
		DataDir: "./data",
	}
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Apply defaults for empty fields
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Server.Port == "" {
		cfg.Server.Port = "3000"
	}
	if cfg.DataDir == "" {
		cfg.DataDir = "./data"
	}

	return cfg, nil
}

// Save writes the current configuration back to the config file.
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
