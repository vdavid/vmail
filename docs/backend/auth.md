# Auth

The `auth` backend feature handles authentication and authorization for the V-Mail API.

The feature set is not in a single package but rather a scattered bunch of files that provide auth.

## Components

* **`internal/api/auth_handler.go`**: HTTP handler for the `/api/v1/auth/status` endpoint.
    * `GetAuthStatus`: Returns authentication and setup status for the current user.
    * Checks if the user has completed onboarding by verifying user settings exist in the database.

* **`internal/auth/middleware.go`**: Authentication middleware.
    * `RequireAuth`: HTTP middleware that validates Bearer tokens in the Authorization header.
    * `ValidateToken`: Validates Authelia JWT tokens and extracts the user's email (currently a stub for development).
    * `GetUserEmailFromContext`: Helper to extract the authenticated user's email from the request context.

* **`internal/db/user.go`**: Database operations for users.
    * `GetOrCreateUser`: Gets or creates a user record by email address.

* **`internal/db/user_settings.go`**: Database operations for user settings.
    * `UserSettingsExist`: Checks if user settings exist for a given user ID.

## Flow

1. The V-Mail front end redirects the user to Authelia for login.
2. After successful login, Authelia provides a session token, a JWT, which the front end stores in the browser.
3. After this, all API requests will include this as a Bearer token in the Authorization header.
4. `RequireAuth` middleware validates the token and extracts the user's email.
5. The email is stored in the request context for use by handlers.
6. Handlers use `GetUserEmailFromContext` to retrieve the authenticated user's email.
7. The auth handler checks if the user has completed setup by querying for user settings.

## Current limitations

* `ValidateToken` is a stub that always returns "test@example.com" in production mode. It must be implemented to
  actually validate Authelia JWT tokens before deployment.
* In test mode (`VMAIL_TEST_MODE=true`), tokens can be prefixed with "email:" to specify the test user email.
