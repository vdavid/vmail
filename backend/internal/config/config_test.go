package config

import (
	"os"
	"testing"
)

func TestNewConfig(t *testing.T) {
	originalEnv := os.Getenv("VMAIL_ENV")
	defer func(key, value string) {
		_ = os.Setenv(key, value)
	}("VMAIL_ENV", originalEnv)

	_ = os.Setenv("VMAIL_ENV", "production")
	_ = os.Setenv("VMAIL_ENCRYPTION_KEY_BASE64", "dGVzdC1rZXktMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM=")
	_ = os.Setenv("AUTHELIA_URL", "http://authelia:9091")
	_ = os.Setenv("VMAIL_DB_PASSWORD", "test-password")
	_ = os.Setenv("VMAIL_DB_HOST", "localhost")
	_ = os.Setenv("VMAIL_DB_PORT", "5432")
	_ = os.Setenv("VMAIL_DB_USER", "test-user")
	_ = os.Setenv("VMAIL_DB_NAME", "testdb")
	_ = os.Setenv("PORT", "3000")

	defer func() {
		_ = os.Unsetenv("VMAIL_ENV")
		_ = os.Unsetenv("VMAIL_ENCRYPTION_KEY_BASE64")
		_ = os.Unsetenv("AUTHELIA_URL")
		_ = os.Unsetenv("VMAIL_DB_PASSWORD")
		_ = os.Unsetenv("VMAIL_DB_HOST")
		_ = os.Unsetenv("VMAIL_DB_PORT")
		_ = os.Unsetenv("VMAIL_DB_USER")
		_ = os.Unsetenv("VMAIL_DB_NAME")
		_ = os.Unsetenv("PORT")
	}()

	config, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() returned error: %v", err)
	}

	if config.Environment != "production" {
		t.Errorf("expected Environment 'production', got '%s'", config.Environment)
	}

	if config.EncryptionKeyBase64 != "dGVzdC1rZXktMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM=" {
		t.Errorf("expected EncryptionKeyBase64 'dGVzdC1rZXktMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM=', got '%s'", config.EncryptionKeyBase64)
	}

	if config.AutheliaURL != "http://authelia:9091" {
		t.Errorf("expected AutheliaURL 'http://authelia:9091', got '%s'", config.AutheliaURL)
	}

	if config.DBHost != "localhost" {
		t.Errorf("expected DBHost 'localhost', got '%s'", config.DBHost)
	}

	if config.DBPort != "5432" {
		t.Errorf("expected DBPort '5432', got '%s'", config.DBPort)
	}

	if config.DBUsername != "test-user" {
		t.Errorf("expected DBUsername 'testuser', got '%s'", config.DBUsername)
	}

	if config.DBPassword != "test-password" {
		t.Errorf("expected DBPassword 'test-password', got '%s'", config.DBPassword)
	}

	if config.DBName != "testdb" {
		t.Errorf("expected DBName 'testdb', got '%s'", config.DBName)
	}

	if config.Port != "3000" {
		t.Errorf("expected Port '3000', got '%s'", config.Port)
	}
}

func TestNewConfigWithDefaults(t *testing.T) {
	_ = os.Setenv("VMAIL_ENV", "production")
	_ = os.Setenv("VMAIL_ENCRYPTION_KEY_BASE64", "test-key")
	_ = os.Setenv("AUTHELIA_URL", "http://authelia:9091")
	_ = os.Setenv("VMAIL_DB_PASSWORD", "password")

	defer func() {
		_ = os.Unsetenv("VMAIL_ENV")
		_ = os.Unsetenv("VMAIL_ENCRYPTION_KEY_BASE64")
		_ = os.Unsetenv("AUTHELIA_URL")
		_ = os.Unsetenv("VMAIL_DB_PASSWORD")
	}()

	config, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() returned error: %v", err)
	}

	if config.DBHost != "localhost" {
		t.Errorf("expected default DBHost 'localhost', got '%s'", config.DBHost)
	}

	if config.DBPort != "5432" {
		t.Errorf("expected default DBPort '5432', got '%s'", config.DBPort)
	}

	if config.DBUsername != "vmail" {
		t.Errorf("expected default DBUsername 'vmail', got '%s'", config.DBUsername)
	}

	if config.DBName != "vmail" {
		t.Errorf("expected default DBName 'vmail', got '%s'", config.DBName)
	}

	if config.Port != "8080" {
		t.Errorf("expected default Port '8080', got '%s'", config.Port)
	}

	if config.Timezone != "UTC" {
		t.Errorf("expected default Timezone 'UTC', got '%s'", config.Timezone)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		shouldErr bool
		errMsg    string
	}{
		{
			name: "valid config",
			config: &Config{
				EncryptionKeyBase64: "dGVzdC1rZXktMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM=",
				AutheliaURL:         "http://authelia:9091",
				DBPassword:          "password",
			},
			shouldErr: false,
		},
		{
			name: "missing encryption key",
			config: &Config{
				AutheliaURL: "http://authelia:9091",
				DBPassword:  "password",
			},
			shouldErr: true,
			errMsg:    "VMAIL_ENCRYPTION_KEY_BASE64 is required",
		},
		{
			name: "missing authelia URL",
			config: &Config{
				EncryptionKeyBase64: "dGVzdC1rZXktMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM=",
				DBPassword:          "password",
			},
			shouldErr: true,
			errMsg:    "AUTHELIA_URL is required",
		},
		{
			name: "missing DB password",
			config: &Config{
				EncryptionKeyBase64: "dGVzdC1rZXktMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM=",
				AutheliaURL:         "http://authelia:9091",
			},
			shouldErr: true,
			errMsg:    "VMAIL_DB_PASSWORD is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.shouldErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
			if tt.shouldErr && err != nil && err.Error() != tt.errMsg {
				t.Errorf("expected error message '%s', got '%s'", tt.errMsg, err.Error())
			}
		})
	}
}

func TestGetDatabaseURL(t *testing.T) {
	config := &Config{
		DBUsername: "test-user",
		DBPassword: "test-password",
		DBHost:     "localhost",
		DBPort:     "5432",
		DBName:     "testdb",
		DBSSLMode:  "disable",
	}

	expected := "postgres://test-user:test-password@localhost:5432/testdb?sslmode=disable"
	got := config.GetDatabaseURL()

	if got != expected {
		t.Errorf("expected database URL '%s', got '%s'", expected, got)
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	_ = os.Setenv("TEST_KEY", "test-value")
	defer func() {
		_ = os.Unsetenv("TEST_KEY")
	}()

	got := getEnvOrDefault("TEST_KEY", "default")
	if got != "test-value" {
		t.Errorf("expected 'test-value', got '%s'", got)
	}

	got = getEnvOrDefault("NONEXISTENT_KEY", "default")
	if got != "default" {
		t.Errorf("expected 'default', got '%s'", got)
	}
}
