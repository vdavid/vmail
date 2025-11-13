package config

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds the application configuration loaded from environment variables.
type Config struct {
	// Environment is the deployment environment (development, production, etc.).
	// Defaults to "development" if VMAIL_ENV is not set.
	Environment string
	// EncryptionKeyBase64 is the base64-encoded encryption key used for encrypting/decrypting
	// user credentials. Must be 32 bytes when decoded (44 characters in base64).
	EncryptionKeyBase64 string
	// AutheliaURL is the base URL of the Authelia authentication server.
	AutheliaURL string
	// DBHost is the PostgreSQL database hostname. Defaults to "localhost".
	DBHost string
	// DBPort is the PostgreSQL database port. Defaults to "5432".
	DBPort string
	// DBUsername is the PostgreSQL database username. Defaults to "vmail".
	DBUsername string
	// DBPassword is the PostgreSQL database password. Required, no default.
	DBPassword string
	// DBName is the PostgreSQL database name. Defaults to "vmail".
	DBName string
	// DBSSLMode is the PostgreSQL SSL mode (disable, require, verify-full, etc.). Defaults to "disable".
	DBSSLMode string
	// Port is the HTTP server port. Defaults to "11764".
	Port string
	// Timezone is the application timezone (e.g., "UTC", "America/New_York"). Defaults to "UTC".
	Timezone string
}

// NewConfig loads and returns a new Config instance from environment variables.
func NewConfig() (*Config, error) {
	env := os.Getenv("VMAIL_ENV")
	if env == "" {
		env = "development"
	}

	if env == "development" {
		if err := godotenv.Load(); err != nil {
			log.Printf("Warning: .env file not found, using environment variables")
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
		Port:                getEnvOrDefault("PORT", "11764"),
		Timezone:            getEnvOrDefault("TZ", "UTC"),
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate checks that all required configuration values are set and valid.
func (c *Config) Validate() error {
	if c.EncryptionKeyBase64 == "" {
		return fmt.Errorf("VMAIL_ENCRYPTION_KEY_BASE64 is required")
	}

	// Validate EncryptionKeyBase64 format: must be valid base64 and decode to 32 bytes
	decoded, err := base64.StdEncoding.DecodeString(c.EncryptionKeyBase64)
	if err != nil {
		return fmt.Errorf("VMAIL_ENCRYPTION_KEY_BASE64 is not valid base64: %w", err)
	}
	if len(decoded) != 32 {
		return fmt.Errorf("VMAIL_ENCRYPTION_KEY_BASE64 must decode to 32 bytes, got %d bytes", len(decoded))
	}

	if c.AutheliaURL == "" {
		return fmt.Errorf("AUTHELIA_URL is required")
	}

	// Validate AutheliaURL format: must be a valid URL with http or https scheme
	parsedURL, err := url.Parse(c.AutheliaURL)
	if err != nil {
		return fmt.Errorf("AUTHELIA_URL is not a valid URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("AUTHELIA_URL must use http:// or https:// scheme, got: %s", parsedURL.Scheme)
	}

	if c.DBPassword == "" {
		return fmt.Errorf("VMAIL_DB_PASSWORD is required")
	}

	// Validate DBPort format: must be a valid port number (1-65535)
	if err := validatePort(c.DBPort); err != nil {
		return fmt.Errorf("VMAIL_DB_PORT is not a valid port number: %w", err)
	}

	// Validate Port format: must be a valid port number (1-65535)
	if err := validatePort(c.Port); err != nil {
		return fmt.Errorf("PORT is not a valid port number: %w", err)
	}

	return nil
}

// validatePort checks if a string represents a valid port number (1-65535).
func validatePort(portStr string) error {
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("port must be a number: %w", err)
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}
	return nil
}

// GetDatabaseURL returns a PostgreSQL connection string built from the configuration.
// The password and username are properly URL-encoded to handle special characters.
func (c *Config) GetDatabaseURL() string {
	// URL-encode username and password to handle special characters
	encodedUsername := url.QueryEscape(c.DBUsername)
	encodedPassword := url.QueryEscape(c.DBPassword)

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		encodedUsername,
		encodedPassword,
		c.DBHost,
		c.DBPort,
		c.DBName,
		c.DBSSLMode,
	)
}

// getEnvOrDefault retrieves an environment variable, returning the default value if not set or empty.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
