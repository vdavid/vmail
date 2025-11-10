# E2E Tests

End-to-end tests for V-Mail using Playwright.

## Current Status

The E2E test infrastructure is set up, but tests currently require manual server setup. Future work will automate server startup with test IMAP/SMTP servers.

## Running Tests

### Prerequisites

1. Backend server running on `http://localhost:8080`
2. Test IMAP server running (using `go-imap/server`)
3. Test SMTP server running (using `go-smtp`)
4. Database running and migrated

### Manual Setup (Current)

For now, you need to manually start:
- Backend server configured to use test IMAP/SMTP servers
- Test IMAP server on a known port
- Test SMTP server on a known port

### Future: Automated Setup

We plan to automate this by:
- Creating a test server binary that starts backend + test servers
- Or using Playwright's `webServer` config to start everything
- Or using a test harness that manages all services

## Running Tests

```bash
# Run all E2E tests
pnpm test:e2e

# Run tests in UI mode
pnpm test:e2e:ui

# Run specific test file
pnpm exec playwright test e2e/tests/onboarding.spec.ts
```

## Test Structure

- `e2e/tests/` - Test files
- `e2e/fixtures/` - Test data and auth helpers
- `e2e/utils/` - Utility functions

