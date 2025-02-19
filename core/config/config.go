package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the server configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Network  NetworkConfig  `yaml:"network"`
	Region   RegionConfig   `yaml:"region"`
	Security SecurityConfig `yaml:"security"`
}

// ServerConfig contains server-specific settings
type ServerConfig struct {
	ID             string   `yaml:"id"`
	ListenAddr     string   `yaml:"listen_addr"`
	ValidatorIDs   []string `yaml:"validator_ids"`
	ValidatorCount int      `yaml:"validator_count"`
}

// NetworkConfig contains P2P network settings
type NetworkConfig struct {
	BootstrapPeers []string `yaml:"bootstrap_peers"`
	BootstrapNodes []string `yaml:"bootstrap_nodes"`
	MaxPeers       int      `yaml:"max_peers"`
	DialTimeout    int      `yaml:"dial_timeout"`
}

// RegionConfig contains region-specific settings
type RegionConfig struct {
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Peers       []string `yaml:"peers"`
}

// SecurityConfig contains security-related settings
type SecurityConfig struct {
	KeyFile     string `yaml:"key_file"`
	KeyPath     string `yaml:"key_path"`
	KeyType     string `yaml:"key_type"`
	TLSEnabled  bool   `yaml:"tls_enabled"`
	TLSCertFile string `yaml:"tls_cert_file"`
	TLSKeyFile  string `yaml:"tls_key_file"`
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			ListenAddr:     ":8080",
			ValidatorCount: 4,
		},
		Network: NetworkConfig{
			MaxPeers:    50,
			DialTimeout: 30,
		},
		Security: SecurityConfig{
			KeyFile: "server.key",
			KeyPath: "server.key",
			KeyType: "ed25519",
		},
	}
}

// LoadConfig loads configuration from a file
func LoadConfig(path string) (*Config, error) {
	// Start with default config
	config := DefaultConfig()

	// Read config file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// SaveConfig saves configuration to a file
func SaveConfig(config *Config, path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.ValidatorCount < 1 {
		return fmt.Errorf("validator count must be at least 1")
	}

	if c.Network.MaxPeers < 1 {
		return fmt.Errorf("max peers must be at least 1")
	}

	if c.Network.DialTimeout < 1 {
		return fmt.Errorf("dial timeout must be at least 1 second")
	}

	if c.Security.TLSEnabled {
		if c.Security.TLSCertFile == "" {
			return fmt.Errorf("TLS cert file required when TLS is enabled")
		}
		if c.Security.TLSKeyFile == "" {
			return fmt.Errorf("TLS key file required when TLS is enabled")
		}
	}

	return nil
}
