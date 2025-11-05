package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment         string
	EncryptionKeyBase64 string
	AutheliaURL         string
	DBHost              string
	DBPort              string
	DBUsername          string
	DBPassword          string
	DBName              string
	DBSSLMode           string
	Port                string
	Timezone            string
}

func NewConfig() (*Config, error) {
	env := os.Getenv("VMAIL_ENV")
	if env == "" {
		env = "development"
	}

	if env == "development" {
		if err := godotenv.Load(); err != nil {
			fmt.Println("Warning: .env file not found, using environment variables")
		}
	}

	config := &Config{
		Environment:         env,
		EncryptionKeyBase64: os.Getenv("VMAIL_ENCRYPTION_KEY_BASE64"),
		AutheliaURL:         os.Getenv("AUTHELIA_URL"),
		DBHost:              getEnvOrDefault("VMAIL_DB_HOST", "localhost"),
		DBPort:              getEnvOrDefault("VMAIL_DB_PORT", "5432"),
		DBUsername:          getEnvOrDefault("VMAIL_DB_USER", "vmail"),
		DBPassword:          os.Getenv("VMAIL_DB_PASSWORD"),
		DBName:              getEnvOrDefault("VMAIL_DB_NAME", "vmail"),
		DBSSLMode:           getEnvOrDefault("VMAIL_DB_SSLMODE", "disable"),
		Port:                getEnvOrDefault("PORT", "8080"),
		Timezone:            getEnvOrDefault("TZ", "UTC"),
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) Validate() error {
	if c.EncryptionKeyBase64 == "" {
		return fmt.Errorf("VMAIL_ENCRYPTION_KEY_BASE64 is required")
	}

	if c.AutheliaURL == "" {
		return fmt.Errorf("AUTHELIA_URL is required")
	}

	if c.DBPassword == "" {
		return fmt.Errorf("VMAIL_DB_PASSWORD is required")
	}

	return nil
}

func (c *Config) GetDatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUsername,
		c.DBPassword,
		c.DBHost,
		c.DBPort,
		c.DBName,
		c.DBSSLMode,
	)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
