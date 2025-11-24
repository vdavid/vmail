# Contributing

Thanks for your interest in contributing to V-Mail! You're most welcome to do so.
The easiest way to contribute is to fork the repo, make your changes, and submit a PR.
This doc is here to help you get started.

(This doc is a WIP. If you have questions, please open an issue.)

## Links


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

### 3. Backend

1. Copy `.env.example` to `.env` in the project root:
   ```bash
   cp .env.example .env
   ```
2. Edit `.env` to match your local setup:
   - Set `VMAIL_DB_HOST`, `VMAIL_DB_PASSWORD`, etc. to point to your Postgres instance.
   - Set `AUTHELIA_URL` to your Authelia instance.
   - Ensure `VMAIL_ENCRYPTION_KEY_BASE64` is set (generate a random 32-byte key and base64 encode it if needed).
3. Run the server:
   ```bash
   go run backend/cmd/server/main.go
   ```
   The server will start on the port defined in `PORT` (default 11764).

### 4. Frontend

1. Navigate to the frontend directory:
   ```bash
   cd frontend
   ```
2. Install dependencies:
   ```bash
   pnpm install
   ```
3. Start the development server:
   ```bash
   pnpm dev
   ```
4. Open the URL shown (usually `http://localhost:5173`).

## Tooling

The project includes several utility scripts in `scripts/`. See [docs/scripts](docs/scripts.md) for their docs.

## Testing
`scripts/check.sh` runs all formatting, linting, and tests. Always run it before committing and ensure all checks pass.
See `./scripts/check.sh --help` to learn about more specific uses.

## Keeping things up to date

For a step-by-step process on updating tools, dependencies, and Docker images, see [`docs/maintenance.md`](docs/maintenance.md).
