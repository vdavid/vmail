# Technical decisions

## Tooling

* **Version Management:** [`mise`](https://mise.jdx.dev) for tool version management.
    * Manages Go, Node.js, and pnpm versions via `mise.toml`.
* **Database Migrations:** [`golang-migrate`](https://github.com/golang-migrate/migrate)
    * SQL migration tool for managing Postgres schema changes.
    * Migrations are stored in `backend/migrations/`.
* **Containerization:** [`Docker`](https://www.docker.com/) and [`Docker Compose`](https://docs.docker.com/compose/)
    * Used for local development and deployment.
    * Multi-stage Dockerfile builds both frontend and backend.
* **Development Process Management (dev-only):** [`Overmind`](https://github.com/DarthSim/overmind)
    * Process manager for running multiple development services concurrently.
    * Uses `tmux` under the hood to provide prefixed logging output.
    * Configured via `Procfile.dev` to run backend (with Air) and frontend (Vite) together.
    * **Note:** This is a development-only tool and is not used in production.
* **Go Live Reload (dev-only):** [`air`](https://github.com/air-verse/air)
    * Live reload tool for Go applications during development.
    * Automatically rebuilds and restarts the server when Go files change.
    * Configured via `.air.toml` in the project root.
    * **Note:** This is a development-only tool and is not used in production.
* **CI/CD:** [`GitHub Actions`](https://github.com/features/actions)
    * Automated testing, linting, and formatting checks on pull requests and pushes.
* **Code Quality (Go):**
    * [`gofmt`](https://pkg.go.dev/cmd/gofmt) (standard library): Code formatting.
    * [`govulncheck`](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck): Security vulnerability scanning.
    * [`go vet`](https://pkg.go.dev/cmd/vet) (standard library): Static analysis.
    * [`staticcheck`](https://staticcheck.io/): Advanced static analysis with additional checks.
    * [`ineffassign`](https://github.com/gordonklaus/ineffassign): Detects ineffective assignments.
    * [`misspell`](https://github.com/client9/misspell): Spell checking in code and comments.
    * [`gocyclo`](https://github.com/fzipp/gocyclo): Cyclomatic complexity checking (warns on functions > 15).
    * [`nilaway`](https://github.com/uber-go/nilaway): Nil pointer analysis.
* **Custom Quality Control:** `scripts/check/` (Go-based tool)
    * Orchestrates all formatting, linting, and testing checks.
    * Auto-fixes issues when not in CI mode.
    * See [`scripts/check/README.md`](../scripts/check/README.md) for details.

## Back end

### Architecture

* **API Style:** REST API with WebSocket support for real-time updates.
    * REST endpoints under `/api/v1` for standard CRUD operations.
    * WebSocket endpoint at `/api/v1/ws` for pushing real-time email notifications.
    * WebSocket Hub manages multiple connections per user (e.g., multiple browser tabs).
    * IMAP IDLE integration triggers incremental syncs and pushes updates via WebSocket.
* **HTTP Router:** Standard library [`http.ServeMux`](https://pkg.go.dev/net/http#ServeMux)
    * Battle-tested and well-documented. No external router dependency needed.
    * Selected based on [this guide](https://www.alexedwards.net/blog/which-go-router-should-i-use)
* **Authentication:** [`Authelia`](https://www.authelia.com) (external service)
    * Self-hosted authentication and authorization server.
    * V-Mail validates JWT tokens from Authelia on each API request.
    * User email is extracted from the token and used for user identification.

### DB

We chose **Postgres** for its robustness, reliability, and excellent support for `JSONB`,
which is useful for flexible payloads like our action queue.

* **Version:** Postgres 14+ (16 recommended for production)
* **Role:** The database serves as a cache and settings store, not a complete copy of the mailbox.
    * Caches thread/message metadata for fast UI rendering.
    * Stores encrypted IMAP/SMTP credentials.
    * Tracks sync state and materialized counts for performance.
* **Migrations:** Managed via `golang-migrate` with SQL files in `backend/migrations/`.

### Go libraries used

* **IMAP Client:** [`github.com/emersion/go-imap`](https://github.com/emersion/go-imap)
    * This seems to be the *de facto* standard library for client-side IMAP in Go.
      It seems well-maintained and supports the necessary extensions like `THREAD`.
* **IMAP Extensions:**
    * [`github.com/emersion/go-imap-idle`](https://github.com/emersion/go-imap-idle): IMAP IDLE extension support for real-time email notifications.
    * [`github.com/emersion/go-imap-sortthread`](https://github.com/emersion/go-imap-sortthread): IMAP SORT and THREAD extension support for sorting and threading emails on the server.
* **MIME Parsing:** [`github.com/jhillyerd/enmime`](https://github.com/jhillyerd/enmime)
    * The Go standard library is not enough for real-world, complex emails.
    * `enmime` robustly handles attachments, encodings,
      and HTML/text parts. [Docs here.](https://pkg.go.dev/github.com/jhillyerd/enmime)
* **SMTP Sending:** [`github.com/emersion/go-smtp`](https://github.com/emersion/go-smtp)
    * SMTP client library for sending emails. Part of the emersion email ecosystem,
      providing a clean API for SMTP operations.
* **HTTP Router:** [`http.ServeMux`](https://pkg.go.dev/net/http#ServeMux)
    * It's part of the Go standard library, is battle-tested and well-documented.
    * Selected based on [this guide](https://www.alexedwards.net/blog/which-go-router-should-i-use)
* **Postgres Driver:** [`github.com/jackc/pgx`](https://github.com/jackc/pgx)
    * The modern, high-performance Postgres driver for Go. We need no full ORM (like [GORM](https://gorm.io/))
      for this project.
* **WebSocket:** [`github.com/gorilla/websocket`](https://github.com/gorilla/websocket)
    * WebSocket implementation for real-time communication between frontend and backend.
    * Used for pushing email updates to connected clients via IMAP IDLE notifications.
* **Configuration:** [`github.com/joho/godotenv`](https://github.com/joho/godotenv)
    * For loading environment variables from `.env` files in development mode.
    * Automatically loads `.env` when `VMAIL_ENV` is set to "development".
* **Encryption:** Standard `crypto/aes` and `crypto/cipher`
    * For encrypting/decrypting user credentials in the DB using AES-GCM.
* **Testing:**
    * [`github.com/stretchr/testify`](https://github.com/stretchr/testify): For assertions and test suites.
        * Provides `assert` and `require` packages for cleaner test assertions.
        * Widely adopted in the Go community and well-maintained.
    * [`github.com/vektra/mockery`](https://github.com/vektra/mockery): For generating mocks from interfaces.
        * Automatically generates mock implementations from Go interfaces, reducing boilerplate.
        * Mocks are generated in `backend/internal/testutil/mocks` and use testify's mock package.
        * See [this guide](https://gist.github.com/maratori/8772fe158ff705ca543a0620863977c2) for rationale on choosing mockery.
    * [`github.com/testcontainers/testcontainers-go`](https://github.com/testcontainers/testcontainers-go): For integration tests with real Postgres containers.

## Front end

### Tech

* **Framework:** React 19+, with functional components and hooks.
* **Language:** [`TypeScript`](https://www.typescriptlang.org/), using no classes, just modules.
* **Build Tool:** [`Vite`](https://vitejs.dev/) with [`@vitejs/plugin-react`](https://github.com/vitejs/vite-plugin-react)
    * Fast build tool and dev server. Provides HMR (Hot Module Replacement) for rapid development.
    * The React plugin enables JSX/TSX transformation and React Fast Refresh.
* **Styling:** [`Tailwind CSS 4`](https://tailwindcss.com/) with [`@tailwindcss/vite`](https://tailwindcss.com/docs/installation/vite)
    * Utility-first CSS framework. The Vite plugin enables seamless integration with the build process.
* **Package manager:** pnpm.
* **State management:**
    * [`@tanstack/react-query`](https://tanstack.com/query) (React Query): For server state (caching, invalidating, and refetching all data from our Go API).
    * [`zustand`](https://github.com/pmndrs/zustand): For simple, global UI state (e.g., current selection, composer open/closed).
* **Routing:** [`react-router-dom`](https://reactrouter.com/) (for URL-based navigation, e.g., `/inbox`, `/thread/id`).
* **Linting/Formatting:**
    * [`ESLint`](https://eslint.org/): Code linting with TypeScript support via [`@typescript-eslint/eslint-plugin`](https://typescript-eslint.io/) and [`@typescript-eslint/parser`](https://typescript-eslint.io/).
    * [`eslint-plugin-react`](https://github.com/jsx-eslint/eslint-plugin-react), [`eslint-plugin-react-hooks`](https://github.com/facebook/react/tree/main/packages/eslint-plugin-react-hooks), and related plugins for React-specific rules.
    * [`eslint-plugin-import`](https://github.com/import-js/eslint-plugin-import): For import/export validation.
    * [`Prettier`](https://prettier.io/): Code formatting with ESLint integration via [`eslint-config-prettier`](https://github.com/prettier/eslint-config-prettier) and [`eslint-plugin-prettier`](https://github.com/prettier/eslint-plugin-prettier).
* **Testing:**
    * [`Vitest`](https://vitest.dev/): Fast unit and integration test runner, compatible with Jest APIs.
    * [`@testing-library/react`](https://testing-library.com/react): For testing React components.
    * [`@testing-library/jest-dom`](https://github.com/testing-library/jest-dom): Custom Jest/Vitest matchers for DOM assertions.
    * [`@testing-library/user-event`](https://testing-library.com/docs/user-event/intro): For simulating user interactions.
    * [`msw`](https://mswjs.io/) (Mock Service Worker): For API mocking in tests.
    * [`@playwright/test`](https://playwright.dev/): For end-to-end tests.
* **Security:** [`dompurify`](https://github.com/cure53/DOMPurify)
    * To sanitize all email HTML content before rendering it with `dangerouslySetInnerHTML`.
      This is a **mandatory** security step.
