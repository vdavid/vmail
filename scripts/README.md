# Scripts

This directory contains utility scripts for development and project management.

## check.sh

The `check.sh` script runs all formatting, linting, and tests.
It exits with a non-zero code if any check fails.

### Usage

```bash
./scripts/check.sh # Run all checks (backend and frontend)
./scripts/check.sh --backend # Run only backend checks
./scripts/check.sh --frontend # Run only frontend checks
./scripts/check.sh --check <check-name> # Run a specific check
./scripts/check.sh --help # Show help, including a list of available checks
```

### What it checks

**Backend (Go):**

- `gofmt` - Code formatting
- `go mod tidy` - Module file tidiness
- `govulncheck` - Security vulnerability scanning
- `go vet` - Static analysis
- `staticcheck` - Advanced static analysis
- `ineffassign` - Ineffective assignments detection
- `misspell` - Spelling errors
- `gocyclo` - Cyclomatic complexity (warns on functions > 15)
- `go test` - Unit and integration tests

**Frontend (TypeScript):**

- `Prettier` - Code formatting
- `ESLint` - Linting
- `pnpm test` - Unit tests

The script automatically installs missing linting tools and uses the Go version specified in `go.mod` (via `GOTOOLCHAIN=auto`).

## Roadmap burndown chart generator tool

It's at `/scripts/roadmap-burndown.go`

Run it by `go run scripts/roadmap-burndown.go`.

The script analyzes the git history of `ROADMAP.md` to generate a CSV burndown chart showing task completion over time.

It:
- takes all commits where `ROADMAP.md` changed
- counts total and completed tasks (supports both `* [ ]` and `- [ ]` checkbox formats)
- aggregates the data by day (uses the latest commit each day)
- fills empty days using the previous day's data
- outputs a CSV format ready to copy-paste [here](https://docs.google.com/spreadsheets/d/1uy7wZSESecJlwaK9AzEotQIzlOYLvi84jPqQVcSeWIo/edit?gid=0#gid=0)

### Output format

The script outputs a CSV with these columns:

- `date` - In YYYY-MM-DD format
- `total_tasks` - Total number of tasks (checked and unchecked)
- `done_tasks` - Number of completed tasks (checked)
- `commit_message` - The first line of the last commit message of the day
