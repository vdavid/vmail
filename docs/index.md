# Architecture

Here are some clues that should help you get started.

## Component interaction diagram

A very high-level overview of the interaction flow between the main parts:

User's browser → Authelia → auth cookie/token → frontend → API (with token) → backend (←→DB) → email server

## Docs

### General

- [API](api.md) – tells you about the API.
- [features](features.md) – describes what V-Mail can do.
- [style guide](style-guide.md) – is **the style guide**. Make sure to read it and re-read it periodically.
- [technical decisions](tech-stack.md)
- [testing](testing.md) – tells you how to test.
- [scripts](scripts.md) – docs for supporting scripts.

### Back-end

- [auth](backend/auth.md) – Auth middleware for Authelia.
- [config](backend/config.md) – Configuration loading and environment variables.
- [crypto](backend/crypto.md) – Encryption logic for sensitive credentials.
- [db](backend/db.md) – Database schema and usage patterns.
- [folders](backend/folders.md) – IMAP folder management and mapping.
- [imap](backend/imap.md) – IMAP client integration and synchronization.
- [search](backend/search.md) – Email search functionality.
- [settings](backend/settings.md) – User settings handling.
- [thread](backend/thread.md) – Single thread view logic.
- [threads](backend/threads.md) – Thread list/inbox view logic.

## Directory structure

```
/
├── /backend/             # Go backend
│   ├── /cmd/             # Main applications
│   │   └── /server/      # The API server entry point
│   ├── /internal/        # Private application code
│   │   ├── /api/         # HTTP Handlers & routing
│   │   ├── /auth/        # Middleware for Authelia JWTs
│   │   ├── /config/      # Config loading
│   │   ├── /crypto/      # Encryption helpers
│   │   ├── /db/          # Database access layer
│   │   ├── /imap/        # IMAP logic & sync
│   │   ├── /models/      # Core domain models
│   │   └── /testutil/    # Test helpers
│   └── /migrations/      # SQL migrations
│
├── /frontend/            # React frontend
│   ├── /src/
│   │   ├── /components/  # Reusable UI components
│   │   ├── /hooks/       # Custom React hooks
│   │   ├── /pages/       # Page components
│   │   └── /store/       # Zustand state management
│   └── /test/            # Unit tests
│
├── /docs/                # Documentation
├── /e2e/                 # Playwright End-to-End tests
└── /scripts/             # Utility scripts (check.sh, etc.)
```

## Back end

The back end is a **Go** application providing a **REST API** for the front end.
It communicates with the IMAP and the SMTP server and uses a **Postgres** database for caching and internal storage.
