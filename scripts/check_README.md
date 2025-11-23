# check.sh

The `check.sh` script runs all formatting, linting, and tests.
It exits with a non-zero code if any check fails.

## Usage

```bash
./scripts/check.sh # Run all checks (backend and frontend)
./scripts/check.sh --backend # Run only backend checks
./scripts/check.sh --frontend # Run only frontend checks
./scripts/check.sh --check <check-name> # Run a specific check
./scripts/check.sh --help # Show help, including a list of available checks
```

## What it checks

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

- `Prettier` - Code formatting (does not fix formatting)
- `ESLint` - Linting (does not fix linting errors)
- `pnpm test` - Unit tests
- `pnpm test:e2e` - End-to-end tests (weirdly runs all tests twice for now if there are failures)

The script automatically installs missing linting tools and uses the Go version specified in `go.mod` (via `GOTOOLCHAIN=auto`).

