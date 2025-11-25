# Contributing

Thanks for your interest in contributing to V-Mail! You're most welcome to do so.
The easiest way to contribute is to fork the repo, make your changes, and submit a PR.
This doc is here to help you get started.

(This doc is a WIP. If you have questions, please open an issue.)

## Development getting started

This setup lets you run the Go backend and the React frontend locally for debugging.
It is different from the ["Running"](README.md#running) section of the main README, which uses Docker Compose for everything.

### 0. Install mise and tools

This project uses [mise](https://mise.jdx.dev) for tool version management. It automatically installs and manages the correct versions of Go, Node, and pnpm.

1. Install mise:
   ```bash
   brew install mise
   ```

   See more alternatives [here](https://mise.jdx.dev/getting-started.html).

2. In the project directory, install all required tools:
   ```bash
   mise install
   ```

   This will install Go, Node, and pnpm. The tools will be automatically available when you're in the project directory.

3. Install golang-migrate separately (it's not available in mise's registry):
   ```bash
   brew install golang-migrate
   ```
   Alternatively, you can install it via Go:
   ```bash
   go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
   ```

4. Install development tools:
   - **Overmind** (process manager for running multiple services):
     ```bash
     brew install overmind
     ```
     Or via Go:
     ```bash
     go install github.com/DarthSim/overmind/v2/cmd/overmind@latest
     ```
   - **Air** (Go live reload):
     ```bash
     go install github.com/air-verse/air@latest
     ```

### 1. Database

1. Run a Postgres v14+ instance and make it available on a port (e.g., 5432).
   You can use Docker for this:
   ```bash
   docker run -d --name vmail-postgres -p 5432:5432 -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=vmail postgres:16-alpine
   ```
2. Run the migrations:
   ```bash
   migrate -path backend/migrations -database "postgres://postgres:postgres@localhost:5432/vmail?sslmode=disable" up
   ```

### 2. Authentication

1. Set up [Authelia](https://www.authelia.com) locally or remotely.
2. Make sure you know its URL (e.g., `http://localhost:9091`).

### 3. Configuration

1. Copy `.env.example` to `.env` in the project root:
   ```bash
   cp .env.example .env
   ```
2. Edit `.env` to match your local setup:
   - Set `VMAIL_DB_HOST`, `VMAIL_DB_PASSWORD`, etc. to point to your Postgres instance.
   - Set `AUTHELIA_URL` to your Authelia instance.
   - Ensure `VMAIL_ENCRYPTION_KEY_BASE64` is set (generate a random 32-byte key and base64 encode it if needed).

### 4. Running the application

#### Quick Start (Recommended)

After setting up the database and Authelia, you can run both the backend and frontend with live reload using a single command:

```bash
overmind start -f Procfile.dev
```

This will:
- Start the Go backend on `http://localhost:11764` with automatic reload on code changes (using [air](https://github.com/air-verse/air))
- Start the React frontend on `http://localhost:7556` (or the port configured in `VITE_PORT`) with hot module replacement (Vite)
- Display prefixed logs from both processes in a single terminal

Press `Ctrl+C` to stop both servers.

**Note:** Overmind uses `tmux` under the hood. You can connect to individual processes if needed:
```bash
overmind connect backend  # Connect to backend process
overmind connect frontend # Connect to frontend process
```

#### Running services separately

If you prefer to run the backend and frontend in separate terminals:

**Backend (with live reload):**
```bash
air
```

Or without live reload:
```bash
go run backend/cmd/server/main.go
```

**Frontend:**
```bash
cd frontend
pnpm install  # First time only
pnpm dev
```

## Tooling

The project includes several utility scripts in `scripts/`. See [docs/scripts](docs/scripts.md) for their docs.

## Testing
`scripts/check.sh` runs all formatting, linting, and tests. Always run it before committing and ensure all checks pass.
See `./scripts/check.sh --help` to learn about more specific uses.

## Keeping things up to date

For a step-by-step process on updating tools, dependencies, and Docker images, see [`docs/maintenance.md`](docs/maintenance.md).
