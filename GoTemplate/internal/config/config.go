package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Config holds all configuration for our program
type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Logger   LoggerConfig   `json:"logger"`
}

// ServerConfig holds all server related configuration
type ServerConfig struct {
	Host         string        `json:"host"`
	Port         string        `json:"port"`
	Address      string        `json:"address"`
	ReadTimeout  time.Duration `json:"readTimeout"`
	WriteTimeout time.Duration `json:"writeTimeout"`
}

// DatabaseConfig holds all database related configuration
type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbName"`
}

// LoggerConfig holds all logger related configuration
type LoggerConfig struct {
	Level string `json:"level"`
	File  string `json:"file"`
}

// Load reads configuration from file
func Load() (*Config, error) {
	configPath := filepath.Join("configs", "config.json")
	file, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	var cfg Config
	if err := json.Unmarshal(file, &cfg); err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}

	// Set the server address from host and port
	cfg.Server.Address = fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)

	return &cfg, nil
}
