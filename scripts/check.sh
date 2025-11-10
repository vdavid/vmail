#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Install linting tools if not found
install_linting_tools() {
    local tools_missing=0
    
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
    
    if [ $tools_missing -eq 1 ]; then
        echo -e "${YELLOW}Note: Make sure $(go env GOPATH)/bin is in your PATH${NC}"
        echo -e "${YELLOW}Add this to your ~/.zshrc or ~/.bashrc:${NC}"
        echo -e "${YELLOW}  export PATH=\$PATH:\$(go env GOPATH)/bin${NC}"
        echo ""
    fi
}

echo "🔍 Running all checks..."
echo ""

# Ensure linting tools are installed
install_linting_tools

# Add GOPATH/bin to PATH if not already there
if [ -n "$(go env GOPATH)" ]; then
    export PATH="$PATH:$(go env GOPATH)/bin"
fi

FAILED=0

# Backend checks
echo "📦 Backend (Go) checks..."
cd backend

echo -n "  • gofmt... "
if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
    echo -e "${RED}FAILED${NC}"
    echo "    Files not formatted:"
    gofmt -s -l . | sed 's/^/      /'
    FAILED=1
else
    echo -e "${GREEN}OK${NC}"
fi

echo -n "  • go vet... "
if ! go vet ./... > /dev/null 2>&1; then
    echo -e "${RED}FAILED${NC}"
    go vet ./...
    FAILED=1
else
    echo -e "${GREEN}OK${NC}"
fi

echo -n "  • staticcheck... "
if ! command_exists staticcheck; then
    echo -e "${RED}FAILED${NC}"
    echo "    staticcheck not found. Run: go install honnef.co/go/tools/cmd/staticcheck@latest"
    echo "    Then ensure $(go env GOPATH)/bin is in your PATH"
    FAILED=1
elif ! staticcheck ./... > /dev/null 2>&1; then
    echo -e "${RED}FAILED${NC}"
    staticcheck ./...
    FAILED=1
else
    echo -e "${GREEN}OK${NC}"
fi

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

echo -n "  • tests... "
if ! go test ./... > /dev/null 2>&1; then
    echo -e "${RED}FAILED${NC}"
    go test ./...
    FAILED=1
else
    echo -e "${GREEN}OK${NC}"
fi

cd ..

# Frontend checks
echo ""
echo "⚛️  Frontend (TypeScript) checks..."
cd frontend

echo -n "  • Prettier... "
if ! pnpm format:check > /dev/null 2>&1; then
    echo -e "${RED}FAILED${NC}"
    pnpm format:check
    FAILED=1
else
    echo -e "${GREEN}OK${NC}"
fi

echo -n "  • ESLint... "
if ! pnpm lint > /dev/null 2>&1; then
    echo -e "${RED}FAILED${NC}"
    pnpm lint
    FAILED=1
else
    echo -e "${GREEN}OK${NC}"
fi

echo -n "  • tests... "
if ! pnpm test --run > /dev/null 2>&1; then
    echo -e "${RED}FAILED${NC}"
    pnpm test --run
    FAILED=1
else
    echo -e "${GREEN}OK${NC}"
fi

cd ..

echo ""
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ All checks passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some checks failed. Please fix the issues above.${NC}"
    exit 1
fi

