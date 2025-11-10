#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Global state
FAILED=0

# Check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Install linting tools if not found
install_linting_tools() {
    local tools_missing=0
    
    # Add GOPATH/bin to PATH if not already there (needed for command_exists to work)
    if [ -n "$(go env GOPATH)" ]; then
        local gopath_bin="$(go env GOPATH)/bin"
        if [[ ":$PATH:" != *":$gopath_bin:"* ]]; then
            export PATH="$PATH:$gopath_bin"
        fi
    fi
    
    # Use automatic toolchain selection to ensure tools are built with the correct Go version
    export GOTOOLCHAIN=auto
    
    # Install staticcheck if it doesn't exist
    if ! command_exists staticcheck; then
        echo -e "${YELLOW}Installing staticcheck...${NC}"
        go install honnef.co/go/tools/cmd/staticcheck@latest
        tools_missing=1
    fi
    
    if ! command_exists ineffassign; then
        echo -e "${YELLOW}Installing ineffassign...${NC}"
        go install github.com/gordonklaus/ineffassign@latest
        tools_missing=1
    fi
    
    if ! command_exists misspell; then
        echo -e "${YELLOW}Installing misspell...${NC}"
        go install github.com/client9/misspell/cmd/misspell@latest
        tools_missing=1
    fi
    
    if ! command_exists gocyclo; then
        echo -e "${YELLOW}Installing gocyclo...${NC}"
        go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
        tools_missing=1
    fi
    
    if ! command_exists govulncheck; then
        echo -e "${YELLOW}Installing govulncheck...${NC}"
        go install golang.org/x/vuln/cmd/govulncheck@latest
        tools_missing=1
    fi
    
    if ! command_exists nilaway; then
        echo -e "${YELLOW}Installing nilaway...${NC}"
        go install go.uber.org/nilaway/cmd/nilaway@latest
        tools_missing=1
    fi
}

# Backend check functions
run_backend_gofmt() {
    echo -n "  • gofmt... "
    if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
        echo -e "${RED}FAILED${NC}"
        echo "    Files not formatted:"
        gofmt -s -l . | sed 's/^/      /'
        FAILED=1
    else
        echo -e "${GREEN}OK${NC}"
    fi
}

run_backend_go_mod_tidy() {
    echo -n "  • go mod tidy... "
    # Check if go.mod or go.sum would change after running go mod tidy
    if [ -f go.mod ]; then
        # Create temporary copies
        cp go.mod go.mod.bak 2>/dev/null || true
        [ -f go.sum ] && cp go.sum go.sum.bak 2>/dev/null || true
        
        # Run go mod tidy
        go mod tidy > /dev/null 2>&1
        
        # Check if files changed
        MOD_CHANGED=0
        SUM_CHANGED=0
        if [ -f go.mod.bak ] && ! diff -q go.mod go.mod.bak > /dev/null 2>&1; then
            MOD_CHANGED=1
        fi
        if [ -f go.sum.bak ] && ! diff -q go.sum go.sum.bak > /dev/null 2>&1; then
            SUM_CHANGED=1
        fi
        
        # Restore backups
        [ -f go.mod.bak ] && mv go.mod.bak go.mod
        [ -f go.sum.bak ] && mv go.sum.bak go.sum
        
        # Report results
        if [ $MOD_CHANGED -eq 1 ] || [ $SUM_CHANGED -eq 1 ]; then
            echo -e "${RED}FAILED${NC}"
            echo "    Module files need tidying. Run: go mod tidy"
            FAILED=1
        else
            echo -e "${GREEN}OK${NC}"
        fi
        
        # Clean up any remaining backups
        rm -f go.mod.bak go.sum.bak
    else
        echo -e "${YELLOW}SKIP${NC} (no go.mod found)"
    fi
}

run_backend_govulncheck() {
    echo -n "  • govulncheck... "
    if ! command_exists govulncheck; then
        echo -e "${RED}FAILED${NC}"
        echo "    govulncheck not found. Run: go install golang.org/x/vuln/cmd/govulncheck@latest"
        echo "    Then ensure $(go env GOPATH)/bin is in your PATH"
        FAILED=1
    elif ! GOTOOLCHAIN=auto govulncheck ./... > /dev/null 2>&1; then
        echo -e "${RED}FAILED${NC}"
        echo "    Vulnerabilities found:"
        GOTOOLCHAIN=auto govulncheck ./...
        FAILED=1
    else
        echo -e "${GREEN}OK${NC}"
    fi
}

run_backend_go_vet() {
    echo -n "  • go vet... "
    if ! go vet ./... > /dev/null 2>&1; then
        echo -e "${RED}FAILED${NC}"
        go vet ./...
        FAILED=1
    else
        echo -e "${GREEN}OK${NC}"
    fi
}

run_backend_staticcheck() {
    echo -n "  • staticcheck... "
    if ! command_exists staticcheck; then
        echo -e "${RED}FAILED${NC}"
        echo "    staticcheck not found. Run: go install honnef.co/go/tools/cmd/staticcheck@latest"
        echo "    Then ensure $(go env GOPATH)/bin is in your PATH"
        FAILED=1
    else
        # Check if staticcheck might need reinstalling due to Go version mismatch
        # We do this by trying to run it on a single file first (faster than ./...)
        # If it fails with version error, reinstall before running full check
        local test_file
        test_file=$(find . -name "*.go" ! -name "*_test.go" | head -1)
        if [ -n "$test_file" ]; then
            local test_output
            test_output=$(staticcheck "$test_file" 2>&1)
            if echo "$test_output" | grep -q "was built with go"; then
                echo -e "${YELLOW}Reinstalling staticcheck (built with wrong Go version)...${NC}"
                GOTOOLCHAIN=auto go install honnef.co/go/tools/cmd/staticcheck@latest
            fi
        fi
        
        # Now run the actual check
        if ! staticcheck ./... > /dev/null 2>&1; then
            echo -e "${RED}FAILED${NC}"
            staticcheck ./...
            FAILED=1
        else
            echo -e "${GREEN}OK${NC}"
        fi
    fi
}

run_backend_ineffassign() {
    echo -n "  • ineffassign... "
    if ! command_exists ineffassign; then
        echo -e "${RED}FAILED${NC}"
        echo "    ineffassign not found. Run: go install github.com/gordonklaus/ineffassign@latest"
        echo "    Then ensure $(go env GOPATH)/bin is in your PATH"
        FAILED=1
    elif ! ineffassign ./... > /dev/null 2>&1; then
        echo -e "${RED}FAILED${NC}"
        ineffassign ./...
        FAILED=1
    else
        echo -e "${GREEN}OK${NC}"
    fi
}

run_backend_misspell() {
    echo -n "  • misspell... "
    if ! command_exists misspell; then
        echo -e "${RED}FAILED${NC}"
        echo "    misspell not found. Run: go install github.com/client9/misspell/cmd/misspell@latest"
        echo "    Then ensure $(go env GOPATH)/bin is in your PATH"
        FAILED=1
    elif ! misspell -error . > /dev/null 2>&1; then
        echo -e "${RED}FAILED${NC}"
        misspell -error .
        FAILED=1
    else
        echo -e "${GREEN}OK${NC}"
    fi
}

run_backend_gocyclo() {
    echo -n "  • gocyclo (complexity > 15, excluding tests)... "
    if ! command_exists gocyclo; then
        echo -e "${YELLOW}SKIP${NC} (gocyclo not found)"
    else
        # Exclude test files from complexity check
        # Find all Go files that are not test files
        COMPLEX=$(find . -name "*.go" ! -name "*_test.go" -exec gocyclo -over 15 {} \; 2>/dev/null | wc -l)
        if [ "$COMPLEX" -gt 0 ]; then
            echo -e "${YELLOW}WARN${NC} ($COMPLEX functions)"
            find . -name "*.go" ! -name "*_test.go" -exec gocyclo -over 15 {} \; 2>/dev/null | head -5 | sed 's/^/      /'
        else
            echo -e "${GREEN}OK${NC}"
        fi
    fi
}

run_backend_nilaway() {
    echo -n "  • nilaway... "
    if ! command_exists nilaway; then
        echo -e "${YELLOW}SKIP${NC} (nilaway not found)"
    elif ! nilaway ./... > /tmp/nilaway.out 2>&1; then
        echo -e "${RED}FAILED${NC}"
        cat /tmp/nilaway.out | sed 's/^/      /'
        FAILED=1
    else
        echo -e "${GREEN}OK${NC}"
    fi
}

run_backend_tests() {
    echo -n "  • tests... "
    if ! go test ./... > /dev/null 2>&1; then
        echo -e "${RED}FAILED${NC}"
        go test ./...
        FAILED=1
    else
        echo -e "${GREEN}OK${NC}"
    fi
}

# Frontend check functions
run_frontend_prettier() {
    echo -n "  • Prettier... "
    if ! pnpm format:check > /dev/null 2>&1; then
        echo -e "${RED}FAILED${NC}"
        pnpm format:check
        FAILED=1
    else
        echo -e "${GREEN}OK${NC}"
    fi
}

run_frontend_eslint() {
    echo -n "  • ESLint... "
    if ! pnpm lint > /dev/null 2>&1; then
        echo -e "${RED}FAILED${NC}"
        pnpm lint
        FAILED=1
    else
        echo -e "${GREEN}OK${NC}"
    fi
}

run_frontend_tests() {
    echo -n "  • tests... "
    if ! pnpm test --run > /dev/null 2>&1; then
        echo -e "${RED}FAILED${NC}"
        pnpm test --run
        FAILED=1
    else
        echo -e "${GREEN}OK${NC}"
    fi
}

# Run all backend checks
run_all_backend_checks() {
    echo "📦 Backend (Go) checks..."
    cd backend
    
    # Use automatic toolchain selection to ensure we use the Go version specified in go.mod
    export GOTOOLCHAIN=auto
    
    run_backend_gofmt
    run_backend_go_mod_tidy
    run_backend_govulncheck
    run_backend_go_vet
    run_backend_staticcheck
    run_backend_ineffassign
    run_backend_misspell
    run_backend_gocyclo
    run_backend_nilaway
    run_backend_tests
    
    cd ..
}

# Run all frontend checks
run_all_frontend_checks() {
    echo ""
    echo "⚛️  Frontend (TypeScript) checks..."
    cd frontend
    
    run_frontend_prettier
    run_frontend_eslint
    run_frontend_tests
    
    cd ..
}

# Show usage
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Run code quality checks for the project.

OPTIONS:
    --backend-only    Run only backend (Go) checks
    --frontend-only   Run only frontend (TypeScript) checks
    --backend         Alias for --backend-only
    --frontend        Alias for --frontend-only
    --check NAME      Run a single check by name (for CI)
    -h, --help        Show this help message

If no options are provided, runs all checks (backend and frontend).

Available check names:
  Backend: gofmt, go-mod-tidy, govulncheck, go-vet, staticcheck, 
           ineffassign, misspell, gocyclo, nilaway, backend-tests
  Frontend: prettier, eslint, frontend-tests

EXAMPLES:
    $0                    # Run all checks
    $0 --backend-only     # Run only backend checks
    $0 --frontend         # Run only frontend checks (alias)
    $0 --check gofmt      # Run only gofmt check
    $0 --check govulncheck # Run only vulnerability check
EOF
}

# Run a single check by name
run_single_check() {
    local check_name="$1"
    
    # Setup for backend checks
    case "$check_name" in
        gofmt|go-mod-tidy|govulncheck|go-vet|staticcheck|ineffassign|misspell|gocyclo|nilaway|backend-tests)
            # Backend check - need to be in backend directory
            cd backend
            export GOTOOLCHAIN=auto
            
            # Install tools if needed (only for checks that need them)
            case "$check_name" in
                staticcheck|ineffassign|misspell|gocyclo|govulncheck|nilaway)
                    install_linting_tools
                    ;;
            esac
            ;;
        prettier|eslint|frontend-tests)
            # Frontend check - need to be in frontend directory
            cd frontend
            ;;
        *)
            echo "Error: Unknown check name: $check_name" >&2
            echo "Run $0 --help to see available checks" >&2
            exit 1
            ;;
    esac
    
    # Map check names to functions
    case "$check_name" in
        gofmt)
            run_backend_gofmt
            ;;
        go-mod-tidy)
            run_backend_go_mod_tidy
            ;;
        govulncheck)
            run_backend_govulncheck
            ;;
        go-vet)
            run_backend_go_vet
            ;;
        staticcheck)
            run_backend_staticcheck
            ;;
        ineffassign)
            run_backend_ineffassign
            ;;
        misspell)
            run_backend_misspell
            ;;
        gocyclo)
            run_backend_gocyclo
            ;;
        nilaway)
            run_backend_nilaway
            ;;
        tests)
            # This should be called as backend-tests or frontend-tests
            # But for backward compatibility, check context
            if [ -f go.mod ]; then
                # Backend tests
                run_backend_tests
            else
                # Frontend tests
                run_frontend_tests
            fi
            ;;
        backend-tests)
            run_backend_tests
            ;;
        frontend-tests)
            run_frontend_tests
            ;;
        prettier)
            run_frontend_prettier
            ;;
        eslint)
            run_frontend_eslint
            ;;
    esac
    
    # Return to original directory
    cd - > /dev/null
}

# Main execution
main() {
    # Parse arguments
    local run_backend=true
    local run_frontend=true
    local single_check=""
    
    case "${1:-}" in
        --backend-only|--backend)
            run_frontend=false
            ;;
        --frontend-only|--frontend)
            run_backend=false
            ;;
        --check)
            if [ -z "${2:-}" ]; then
                echo "Error: --check requires a check name" >&2
                echo "Run $0 --help to see available checks" >&2
                exit 1
            fi
            single_check="$2"
            run_backend=false
            run_frontend=false
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        "")
            # No arguments, run everything
            ;;
        *)
            echo "Error: Unknown option: $1" >&2
            echo ""
            show_usage
            exit 1
            ;;
    esac
    
    # If running a single check, do that and exit
    if [ -n "$single_check" ]; then
        run_single_check "$single_check"
        exit $FAILED
    fi
    
    echo "🔍 Running all checks..."
    echo ""
    
    # Ensure linting tools are installed (only needed for backend)
    if [ "$run_backend" = true ]; then
        install_linting_tools
    fi
    
    # Run checks
    if [ "$run_backend" = true ]; then
        run_all_backend_checks
    fi
    
    if [ "$run_frontend" = true ]; then
        run_all_frontend_checks
    fi
    
    # Final summary
    echo ""
    if [ $FAILED -eq 0 ]; then
        echo -e "${GREEN}✅ All checks passed!${NC}"
        exit 0
    else
        echo -e "${RED}❌ Some checks failed. Please fix the issues above.${NC}"
        exit 1
    fi
}

# Run main function
main "$@"
