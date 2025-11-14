# Config

The `config` package handles loading and validating application configuration from environment variables.

## Components

* **`internal/config/config.go`**: Configuration loading and validation.
    * `Config`: Struct holding all application configuration values.
    * `NewConfig`: Loads configuration from environment variables, with support for `.env` file in development mode.
    * `Validate`: Validates that all required configuration values are set.
    * `GetDatabaseURL`: Builds a PostgreSQL connection string from database configuration.
    * `getEnvOrDefault`: Helper function to get environment variables with default values.

## Configuration values

### Required

* `VMAIL_ENCRYPTION_KEY_BASE64`: Base64-encoded encryption key (32 bytes when decoded).
* `AUTHELIA_URL`: Base URL of the Authelia authentication server.
* `VMAIL_DB_PASSWORD`: PostgreSQL database password.

### Optional (with defaults)

* `VMAIL_ENV`: Deployment environment (defaults to "development").
* `VMAIL_DB_HOST`: Database hostname (defaults to "localhost").
* `VMAIL_DB_PORT`: Database port (defaults to "5432").
* `VMAIL_DB_USER`: Database username (defaults to "vmail").
* `VMAIL_DB_NAME`: Database name (defaults to "vmail").
* `VMAIL_DB_SSLMODE`: SSL mode (defaults to "disable").
* `PORT`: HTTP server port (defaults to "11764").
* `TZ`: Application timezone (defaults to "UTC").

## Development mode

* When `VMAIL_ENV` is "development" (or not set), the package attempts to load a `.env` file using `godotenv`.
* If the `.env` file is not found, it falls back to environment variables with a warning message.

## Current limitations

* None - all identified issues have been addressed.
