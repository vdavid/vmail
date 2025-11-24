# Maintenance

How to keep V-Mail's tools, dependencies, and infra up to date.

This is an end-to-end process you can follow from top to bottom whenever you plan a maintenance round.  
Before you start, quickly skim `AGENTS.md` and `docs/tech-stack.md` so you know the current stack and rules.

## 1. Decide scope and create a branch

Pick what you want to update. Keep the scope tight so each change is easy to review and roll back if needed.

- **Typical scopes**
  - Go toolchain and backend Go modules.
  - Frontend toolchain (TypeScript, Vite, ESLint, Vitest) and selected frontend deps.
  - Docker images and Postgres.
- **Create a branch**
  - Example: `chore/update-go-1-25`, `chore/update-frontend-tooling`, or `chore/update-docker-postgres`.

## 2. Update Go and backend dependencies

If your scope does not include Go or the backend, skip this section.

### 2.1 Update the Go toolchain

- **Change versions in the right places**
  - `mise.toml`: Update the Go version.
  - `backend/go.mod`: Update the `go` directive.
  - `Dockerfile`: Update the Go image tag used for building and/or running the backend.
  - GitHub Actions: Update the Go version if it is pinned in CI workflows.
- **Install and tidy**
  - From the repo root: `mise install`.
  - From `backend`: run `go mod tidy`.

### 2.2 Update backend Go modules

- **See what is outdated**
  - From `backend`: run `go list -m -u all` to list modules with available updates.
- **Update in small batches**
  - Prefer updating a few related modules at a time rather than everything at once.
  - For important modules like `pgx`, `go-imap`, and `gorilla/websocket`, read their release notes.
  - For each module (or small group), run `go get module@version` and then `go mod tidy`.

## 3. Update frontend tooling and dependencies

If your scope does not include the frontend, skip this section.

### 3.1 Update the frontend toolchain

- **Check what is outdated**
  - From `frontend`: run `pnpm outdated`.
- **Update core tooling first**
  - Update `typescript`, Vite, React tooling, testing tools (Vitest, Testing Library), and linting tools (ESLint, Prettier and related plugins).
  - Edit `frontend/package.json`, then run `pnpm install`.
  - Adjust `tsconfig*.json` or ESLint config only if new versions require it.

### 3.2 Update application dependencies

- **Update in logical groups**
  - Group by role: routing, state management, HTTP, UI, WebSocket client, and so on.
  - For core libraries that affect behavior widely (for example routing, state management, WebSocket, DOMPurify), handle them one by one and read their changelogs.
- **Apply updates**
  - Edit `frontend/package.json` for the selected packages, then run `pnpm install`.

## 4. Update Docker, Postgres, and external services

If your scope does not include containers or external services, skip this section.

- **Docker images**
  - Update image tags in `Dockerfile` (Go and Node images, if any).
  - Update image tags in `docker-compose.yaml` (for example Postgres).
  - Rebuild images with Docker Compose.
- **Postgres**
  - Confirm the target Postgres version in `docs/tech-stack.md`.
  - Update image tags and run migrations against the new version.
- **External services (for example Authelia)**
  - Review their release notes and confirm that authentication flows and JWT claims remain compatible.

## 5. Run automated checks

After you update tools, dependencies, or images, always run checks before you move on.

- **Primary check script**
  - From the repo root: run `./scripts/check.sh`.
- **Focused runs (optional)**
  - When you are iterating quickly, it can help to run only the relevant subset:
    - `./scripts/check.sh --backend` after backend-only work.
    - `./scripts/check.sh --frontend` after frontend-only work.
  - Before opening a pull request, always run the full `./scripts/check.sh`.

Address any errors and warnings, including gocyclo complexity issues, before you continue.

## 6. Do a manual smoke test

Automated checks are important, but a short manual test helps you catch integration issues.

Start the app (for example with Overmind and `Procfile.dev`) and then:

- Sign in through Authelia.
- Load the inbox and paginate through a few pages.
- Open a thread and scroll through it.
- Use search at least once.
- Open the composer, send an email to yourself, and verify it appears in Sent and in your inbox.
- With two browser tabs open, perform an action such as marking a thread as read in one tab and confirm the other tab updates (WebSocket).

## 7. Update docs and open a pull request

Keep the docs and tooling overview in sync with your changes.

- **Docs to consider updating**
  - `AGENTS.md` when you change tools or maintenance expectations.
  - `docs/tech-stack.md` when you change major versions or important libraries.
  - `CONTRIBUTING.md` when you change local setup, commands, or required tools.
- **Open a PR**
  - Summarize what you updated (tools, modules, images).
  - List which automated checks you ran.
  - Mention any manual smoke tests you performed.


