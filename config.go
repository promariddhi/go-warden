package main

import (
	"io"
	"os"

	"github.com/goccy/go-yaml"
)

func LoadConfig() (Config, error) {
	config, err := os.Open("config.yml")
	if err != nil {
		return Config{}, err
	}

	defer func() {
		_ = config.Close()
	}()

	config_b, err := io.ReadAll(config)
	if err != nil {
		return Config{}, err
	}

	c := Config{}

	if err := yaml.Unmarshal(config_b, &c); err != nil {
		return Config{}, err
	}

	return c, nil

}

type Config struct {
	LocalAddress  string `yaml:"local_address"`
	RemoteAddress string `yaml:"remote_address"`
}
