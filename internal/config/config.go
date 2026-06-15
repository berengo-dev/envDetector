// Package config loads and parses the .env-doctor.yaml configuration file.
package config

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

// Config is the root structure of an .env-doctor.yaml file.
type Config struct {
	Version string            `mapstructure:"version"`
	Tools   map[string]string `mapstructure:"tools"`
	Env     []string          `mapstructure:"env"`
	Files   []string          `mapstructure:"files"`
	Ports   map[int]string    `mapstructure:"ports"`
}

// Load reads the environment doctor configuration from the given path. If path
// is empty, it looks for ".env-doctor.yaml" in the current directory.
func Load(path string) (Config, error) {
	v := viper.New()

	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName(".env-doctor")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
	}

	if err := v.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg, func(dc *mapstructure.DecoderConfig) {
		dc.WeaklyTypedInput = true
	}); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}
