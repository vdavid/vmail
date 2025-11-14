# Settings

The `settings` feature provides the user a way to save their settings and preferences, including
their IMAP/SMTP credentials and application preferences.

## Components

* **`internal/api/settings_handler.go`**: HTTP handlers for the `/api/v1/settings` endpoint.
    * `GetSettings`: Returns user settings for the current user (passwords are never included, only a boolean indicating if they're set).
    * `PostSettings`: Saves or updates user settings. Passwords are optional on update (empty passwords preserve existing ones), but required for initial setup.
    * `validateSettingsRequest`: Validates that all required fields are present in the request.

* **`internal/db/user_settings.go`**: Database operations for user settings.
    * `GetUserSettings`: Retrieves user settings by user ID.
    * `SaveUserSettings`: Saves or updates user settings (uses ON CONFLICT for upsert).
    * `UserSettingsExist`: Checks if user settings exist for a given user ID.

## Flow (GetSettings)

1. Handler extracts user ID from request context.
2. Retrieves user settings from the database.
3. Returns 404 if settings don't exist.
4. Builds response without passwords (only indicates if they're set).
5. Returns settings as JSON.

## Flow (PostSettings)

1. Handler extracts user ID from request context.
2. Decodes and validates the request body.
3. Retrieves existing settings (if any) to preserve passwords.
4. Handles password encryption:
    * If password is provided: encrypts and uses the new password.
    * If password is empty and settings exist: preserves existing encrypted password.
    * If password is empty and no settings exist: returns 400 (password required for initial setup).
5. Saves settings to the database.
6. Returns success response.

## Security

* Passwords are encrypted using AES-GCM before storage in the database.
* Passwords are never returned in API responses (only a boolean indicating if they're set).
* Passwords can be updated without re-entering other passwords.

## Error handling

* Returns 404 if settings are not found (GetSettings).
* Returns 400 for validation errors (missing required fields, empty passwords on initial setup).
* Returns 500 for database or encryption errors.
