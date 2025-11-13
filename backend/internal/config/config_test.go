package config

import (
	"net/url"
	"os"
	"strings"
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
		t.Errorf("expected DBUsername 'test-user', got '%s'", config.DBUsername)
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
	_ = os.Setenv("VMAIL_ENCRYPTION_KEY_BASE64", "dGVzdC1rZXktMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM=")
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

	if config.Port != "11764" {
		t.Errorf("expected default Port '11764', got '%s'", config.Port)
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
				DBPort:              "5432",
				Port:                "11764",
			},
			shouldErr: false,
		},
		{
			name: "missing encryption key",
			config: &Config{
				AutheliaURL: "http://authelia:9091",
				DBPassword:  "password",
				DBPort:      "5432",
				Port:        "11764",
			},
			shouldErr: true,
			errMsg:    "VMAIL_ENCRYPTION_KEY_BASE64 is required",
		},
		{
			name: "missing authelia URL",
			config: &Config{
				EncryptionKeyBase64: "dGVzdC1rZXktMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM=",
				DBPassword:          "password",
				DBPort:              "5432",
				Port:                "11764",
			},
			shouldErr: true,
			errMsg:    "AUTHELIA_URL is required",
		},
		{
			name: "missing DB password",
			config: &Config{
				EncryptionKeyBase64: "dGVzdC1rZXktMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM=",
				AutheliaURL:         "http://authelia:9091",
				DBPort:              "5432",
				Port:                "11764",
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
	t.Run("basic URL generation", func(t *testing.T) {
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
	})

	t.Run("handles special characters in password", func(t *testing.T) {
		config := &Config{
			DBUsername: "test-user",
			DBPassword: "p@ss:w/rd%test#",
			DBHost:     "localhost",
			DBPort:     "5432",
			DBName:     "testdb",
			DBSSLMode:  "disable",
		}

		got := config.GetDatabaseURL()
		// The password should be URL-encoded
		if !strings.Contains(got, "p%40ss%3Aw%2Frd%25test%23") {
			t.Errorf("Expected password to be URL-encoded in database URL, got: %s", got)
		}
		// Verify the URL can be parsed
		if _, err := url.Parse(got); err != nil {
			t.Errorf("Generated database URL is not valid: %v", err)
		}
	})

	t.Run("handles special characters in username", func(t *testing.T) {
		config := &Config{
			DBUsername: "user@domain",
			DBPassword: "password",
			DBHost:     "localhost",
			DBPort:     "5432",
			DBName:     "testdb",
			DBSSLMode:  "disable",
		}

		got := config.GetDatabaseURL()
		// The username should be URL-encoded
		if !strings.Contains(got, "user%40domain") {
			t.Errorf("Expected username to be URL-encoded in database URL, got: %s", got)
		}
		// Verify the URL can be parsed
		if _, err := url.Parse(got); err != nil {
			t.Errorf("Generated database URL is not valid: %v", err)
		}
	})
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

func TestNewConfigWithEnvFile(t *testing.T) {
	originalEnv := os.Getenv("VMAIL_ENV")
	defer func(key, value string) {
		_ = os.Setenv(key, value)
	}("VMAIL_ENV", originalEnv)

	_ = os.Setenv("VMAIL_ENV", "development")
	_ = os.Setenv("VMAIL_ENCRYPTION_KEY_BASE64", "dGVzdC1rZXktMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM=")
	_ = os.Setenv("AUTHELIA_URL", "http://authelia:9091")
	_ = os.Setenv("VMAIL_DB_PASSWORD", "test-password")

	defer func() {
		_ = os.Unsetenv("VMAIL_ENV")
		_ = os.Unsetenv("VMAIL_ENCRYPTION_KEY_BASE64")
		_ = os.Unsetenv("AUTHELIA_URL")
		_ = os.Unsetenv("VMAIL_DB_PASSWORD")
	}()

	// Note: This test verifies that NewConfig works in development mode.
	// The actual .env file loading is tested implicitly - if godotenv.Load() fails,
	// it logs a warning but continues (which is acceptable behavior).
	config, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() returned error: %v", err)
	}

	if config.Environment != "development" {
		t.Errorf("expected Environment 'development', got '%s'", config.Environment)
	}
}

func TestValidateEncryptionKey(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		shouldErr bool
		errMsg    string
	}{
		{
			name:      "valid 32-byte base64 key",
			key:       "dGVzdC1rZXktMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM=",
			shouldErr: false,
		},
		{
			name:      "invalid base64",
			key:       "not-valid-base64!!!",
			shouldErr: true,
			errMsg:    "VMAIL_ENCRYPTION_KEY_BASE64 is not valid base64",
		},
		{
			name:      "wrong length (too short)",
			key:       "dGVzdA==", // "test" in base64, only 4 bytes
			shouldErr: true,
			errMsg:    "VMAIL_ENCRYPTION_KEY_BASE64 must decode to 32 bytes",
		},
		{
			name:      "wrong length (too long)",
			key:       "dGVzdC1rZXktMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM=", // 64 bytes
			shouldErr: true,
			errMsg:    "VMAIL_ENCRYPTION_KEY_BASE64 must decode to 32 bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				EncryptionKeyBase64: tt.key,
				AutheliaURL:         "http://authelia:9091",
				DBPassword:          "password",
				DBPort:              "5432",
				Port:                "11764",
			}

			err := config.Validate()
			if tt.shouldErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
			if tt.shouldErr && err != nil && !contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error message to contain '%s', got '%s'", tt.errMsg, err.Error())
			}
		})
	}
}

func TestValidateAutheliaURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		shouldErr bool
		errMsg    string
	}{
		{
			name:      "valid HTTP URL",
			url:       "http://authelia:9091",
			shouldErr: false,
		},
		{
			name:      "valid HTTPS URL",
			url:       "https://authelia.example.com",
			shouldErr: false,
		},
		{
			name:      "invalid URL (wrong scheme)",
			url:       "authelia:9091",
			shouldErr: true,
			errMsg:    "AUTHELIA_URL must use http:// or https:// scheme",
		},
		{
			name:      "invalid URL (path only)",
			url:       "/path/to/authelia",
			shouldErr: true,
			errMsg:    "AUTHELIA_URL must use http:// or https:// scheme",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				EncryptionKeyBase64: "dGVzdC1rZXktMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM=",
				AutheliaURL:         tt.url,
				DBPassword:          "password",
				DBPort:              "5432",
				Port:                "11764",
			}

			err := config.Validate()
			if tt.shouldErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
			if tt.shouldErr && err != nil && !contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error message to contain '%s', got '%s'", tt.errMsg, err.Error())
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	tests := []struct {
		name      string
		dbPort    string
		port      string
		shouldErr bool
		errMsg    string
	}{
		{
			name:      "valid ports",
			dbPort:    "5432",
			port:      "11764",
			shouldErr: false,
		},
		{
			name:      "invalid DBPort (not a number)",
			dbPort:    "not-a-port",
			port:      "11764",
			shouldErr: true,
			errMsg:    "VMAIL_DB_PORT is not a valid port number",
		},
		{
			name:      "invalid Port (not a number)",
			dbPort:    "5432",
			port:      "not-a-port",
			shouldErr: true,
			errMsg:    "PORT is not a valid port number",
		},
		{
			name:      "invalid DBPort (too low)",
			dbPort:    "0",
			port:      "11764",
			shouldErr: true,
			errMsg:    "VMAIL_DB_PORT is not a valid port number",
		},
		{
			name:      "invalid DBPort (too high)",
			dbPort:    "65536",
			port:      "11764",
			shouldErr: true,
			errMsg:    "VMAIL_DB_PORT is not a valid port number",
		},
		{
			name:      "invalid Port (too low)",
			dbPort:    "5432",
			port:      "0",
			shouldErr: true,
			errMsg:    "PORT is not a valid port number",
		},
		{
			name:      "invalid Port (too high)",
			dbPort:    "5432",
			port:      "65536",
			shouldErr: true,
			errMsg:    "PORT is not a valid port number",
		},
		{
			name:      "valid boundary ports",
			dbPort:    "1",
			port:      "65535",
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				EncryptionKeyBase64: "dGVzdC1rZXktMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM=",
				AutheliaURL:         "http://authelia:9091",
				DBPassword:          "password",
				DBPort:              tt.dbPort,
				Port:                tt.port,
			}

			err := config.Validate()
			if tt.shouldErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
			if tt.shouldErr && err != nil && !contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error message to contain '%s', got '%s'", tt.errMsg, err.Error())
			}
		})
	}
}

// contains checks if a string contains a substring (case-sensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
