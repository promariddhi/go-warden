package config

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/goccy/go-yaml"
)

type Config struct {
	LocalAddress         string       `yaml:"local_address"`
	RemoteAddress        string       `yaml:"remote_address"`
	ConnectionLimit      int64        `yaml:"connection_limit"`
	PerIPConnectionLimit int64        `yaml:"per_ip_connection_limit"`
	IdleTimeoutSeconds   int64        `yaml:"idle_timeout_secs"`
	RateLimiter          RateLimiterC `yaml:"rate_limiter"`
}

type RateLimiterC struct {
	TokenBucketLimiter TokenBucketLimiterC `yaml:"token_bucket_limiter"`
}

type TokenBucketLimiterC struct {
	Rate     int64 `yaml:"rate"`
	Capacity int64 `yaml:"capacity"`
}

type ProxyConfig struct {
	LocalAddress       string
	RemoteAddress      string
	IdleTimeoutSeconds int64
}

type ConnectionConfig struct {
	ConnectionLimit      int64
	PerIPConnectionLimit int64
}

type RateLimiterConfig struct {
	RateLimiter RateLimiterC
}

func (c *Config) SplitConfig() (*ProxyConfig, *ConnectionConfig, *RateLimiterConfig) {
	return &ProxyConfig{
			LocalAddress:       c.LocalAddress,
			RemoteAddress:      c.RemoteAddress,
			IdleTimeoutSeconds: c.IdleTimeoutSeconds,
		},
		&ConnectionConfig{
			ConnectionLimit:      c.ConnectionLimit,
			PerIPConnectionLimit: c.PerIPConnectionLimit,
		},
		&RateLimiterConfig{
			RateLimiter: c.RateLimiter,
		}
}

func LoadConfig() (Config, error) {
	path, err := resolveConfig()
	if err != nil {
		return Config{}, err
	}
	config, err := os.Open(path)
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

func resolveConfig() (string, error) {
	if p := flag.Lookup("config"); p != nil {
		if v := p.Value.String(); v != "" {
			return v, nil
		}
	}

	candidates := []string{
		"./config.yml",
		"/etc/db_firewall/config.yml",
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}

	return "", fmt.Errorf("no config file found")
}

func ValidateConfig(cfg Config) error {
	if cfg.LocalAddress == "" {
		return fmt.Errorf("local_address must be set")
	}
	if cfg.RemoteAddress == "" {
		return fmt.Errorf("remote_address must be set")
	}
	if cfg.LocalAddress == cfg.RemoteAddress {
		return fmt.Errorf("local_address and remote_address must not be the same")
	}

	if _, err := net.ResolveTCPAddr("tcp", cfg.LocalAddress); err != nil {
		return fmt.Errorf("invalid local_address: %w", err)
	}
	if _, err := net.ResolveTCPAddr("tcp", cfg.RemoteAddress); err != nil {
		return fmt.Errorf("invalid remote_address: %w", err)
	}

	if cfg.ConnectionLimit <= 0 {
		return fmt.Errorf("connection_limit must be > 0")
	}
	if cfg.PerIPConnectionLimit <= 0 {
		return fmt.Errorf("per_ip_connection_limit must be > 0")
	}
	if cfg.PerIPConnectionLimit > cfg.ConnectionLimit {
		return fmt.Errorf("per_ip_connection_limit cannot exceed connection_limit")
	}

	if cfg.IdleTimeoutSeconds < 0 {
		return fmt.Errorf("idle_timeout_seconds must be >= 0")
	}
	if cfg.IdleTimeoutSeconds > 0 && cfg.IdleTimeoutSeconds < 1 {
		return fmt.Errorf("idle_timeout_seconds must be >= 1 when enabled")
	}

	return nil
}
