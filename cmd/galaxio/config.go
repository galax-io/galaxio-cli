package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/galax-io/galaxio-cli/internal/templatecatalog"
	"gopkg.in/yaml.v3"
)

const configEnv = "GALAXIO_CONFIG"

type config struct {
	Template templateConfig `yaml:"template"`
}

type templateConfig struct {
	Registry string `yaml:"registry"`
}

func configPath() (string, error) {
	if path := os.Getenv(configEnv); path != "" {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".galaxio", "config.yaml"), nil
}

func loadConfig() (config, error) {
	path, err := configPath()
	if err != nil {
		return config{}, err
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return config{}, nil
		}
		return config{}, err
	}

	var cfg config
	if err := yaml.Unmarshal(payload, &cfg); err != nil {
		return config{}, err
	}
	return cfg, nil
}

func saveConfig(cfg config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o644)
}

func resolveTemplateRegistry(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	cfg, err := loadConfig()
	if err != nil {
		return "", fmt.Errorf("load config: %w", err)
	}
	if cfg.Template.Registry != "" {
		return cfg.Template.Registry, nil
	}
	return templatecatalog.DefaultRegistrySource, nil
}
