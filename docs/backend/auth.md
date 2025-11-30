# Auth

V-Mail uses [Authelia](https://www.authelia.com) for authentication. This doc covers how auth works in the backend.

## Deployment options

V-Mail supports two ways to set up authentication:

1. **Bundled Authelia** - V-Mail runs its own Authelia instance via `docker-compose.with-authelia.yml`. 
   Easiest for users who don't have existing auth infrastructure.

2. **Bring your own** - Use an existing Authelia (or any forward_auth-compatible solution like Authentik).
   Just configure your reverse proxy to do `forward_auth` and set the right headers.

See the [README](../../README.md) for setup instructions.

## Auth modes

V-Mail supports two authentication modes, controlled by the `VMAIL_AUTH_MODE` environment variable:

### Dev mode (`VMAIL_AUTH_MODE=dev`)

This is the default for local development. The backend uses stub authentication that always returns
`test@example.com` as the authenticated user. No real authentication happens.

In this mode:
- API requests need a Bearer token in the Authorization header (any non-empty value works)
- WebSocket connections need a `?token=...` query parameter
- `AUTHELIA_URL` is not required

### Header mode (`VMAIL_AUTH_MODE=header`)

This is for production with Authelia. The backend trusts `Remote-User` and `Remote-Email` headers set by the
reverse proxy (Caddy/Nginx) after Authelia validates the user's session.

In this mode:
- API requests don't need an Authorization header (the reverse proxy handles auth)
- WebSocket upgrade requests get headers from the reverse proxy too
- `AUTHELIA_URL` is required (so the frontend knows where to redirect on 401)

## Components

* **`internal/auth/middleware.go`**: Authentication middleware.
    * `RequireAuth`: HTTP middleware that authenticates requests based on auth mode.
    * In dev mode: Validates Bearer tokens from the Authorization header.
    * In header mode: Reads `Remote-Email` or `Remote-User` headers from the reverse proxy.
    * `GetUserEmailFromContext`: Helper to get the authenticated user's email from the request context.

* **`internal/api/ws_handler.go`**: WebSocket handler with its own auth.
    * In dev mode: Validates token from query parameter or Authorization header.
    * In header mode: Reads `Remote-Email` or `Remote-User` headers from the upgrade request.

* **`internal/api/auth_handler.go`**: HTTP handler for `/api/v1/auth/status`.
    * Returns whether the user has completed onboarding (has settings configured).

* **`internal/db/user.go`**: Database operations for users.
    * `GetOrCreateUser`: Gets or creates a user record by email address.

## Flow

### Production (header mode)

1. User navigates to V-Mail.
2. Reverse proxy (Caddy) intercepts the request and calls Authelia's `/api/verify`.
3. If not authenticated, Authelia redirects to its login page.
4. After login, Authelia sets a session cookie.
5. On subsequent requests, Caddy's `forward_auth` verifies the session and adds `Remote-User`/`Remote-Email` headers.
6. V-Mail backend reads these headers and trusts them (since only the reverse proxy can set them).
7. The email is stored in the request context for handlers to use.

### Development (dev mode)

1. Frontend sends requests with `Authorization: Bearer token`.
2. Backend validates the token (stub: always succeeds for non-empty tokens).
3. Backend returns `test@example.com` as the authenticated user.

## Security notes

**Header trust**: In header mode, the backend trusts `Remote-*` headers completely. This is safe because:
- V-Mail should only be accessible through the reverse proxy
- External requests can't set these headers (Caddy/Nginx strips them)
- Never expose V-Mail directly to the internet without the reverse proxy

**WebSocket auth**: WebSocket connections go through the same reverse proxy, so the upgrade request includes
the auth headers. No token is passed in the URL in production.
