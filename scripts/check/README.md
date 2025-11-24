# Check tool

Run it by `./scripts/check.sh` or `./scripts/check/check`.

The tool runs all formatting, linting, and tests for the entire codebase. It exits with a non-zero code if any check fails.

It:
- runs formatting checks (gofmt, prettier) and automatically fixes issues unless `--ci` is used
- runs linting checks (ESLint, staticcheck, etc.) and automatically fixes issues unless `--ci` is used
- runs all tests (backend unit tests, frontend unit tests, E2E tests)
- displays execution time for each check and total runtime
- automatically installs missing linting tools
- uses the Go version specified in `go.mod` (via `GOTOOLCHAIN=auto`)

### Usage

```bash
./scripts/check.sh                    # Run all checks (backend and frontend)
./scripts/check.sh --backend          # Run only backend checks
./scripts/check.sh --frontend          # Run only frontend checks
./scripts/check.sh --check <name>      # Run a specific check
./scripts/check.sh --ci               # Disable auto-fixing (for CI)
./scripts/check.sh --verbose           # Show detailed output
./scripts/check.sh --help              # Show help message
```

### What it checks

**Backend (Go) - Applied to `backend/` and `scripts/`:**

- `gofmt` - Code formatting (auto-fixes if not `--ci`)
- `go mod tidy` - Module file tidiness
- `govulncheck` - Security vulnerability scanning
- `go vet` - Static analysis
- `staticcheck` - Advanced static analysis
- `ineffassign` - Ineffective assignments detection
- `misspell` - Spelling errors
- `gocyclo` - Cyclomatic complexity (warns on functions > 15, doesn't fail)
- `nilaway` - Nil pointer analysis
- `backend-tests` - Unit and integration tests

**Frontend (TypeScript) - Applied to `frontend/` and `frontend/e2e/`:**

- `prettier` - Code formatting (auto-fixes if not `--ci`)
- `eslint` - Linting (auto-fixes if not `--ci`)
- `frontend-tests` - Unit tests
- `e2e-tests` - End-to-end tests

### Available check names

When using `--check <name>`, you can specify:

**Backend checks:**
- `gofmt`
- `go-mod-tidy`
- `govulncheck`
- `go-vet`
- `staticcheck`
- `ineffassign`
- `misspell`
- `gocyclo`
- `nilaway`
- `backend-tests`

**Frontend checks:**
- `prettier`
- `eslint`
- `frontend-tests`
- `e2e-tests`

Check names are case-insensitive.

### Output format

Each check displays its execution time in the format: `OK (123ms)` or `FAILED (1.23s)`. After all checks complete, the total runtime is displayed: `⏱️  Total runtime: 10.47s`.
