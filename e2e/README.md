# E2E Tests

End-to-end tests for V-Mail using Playwright.

## Current Status

✅ **Automated setup is complete!** The test server (`backend/cmd/test-server`) automatically starts all required services.

## Running Tests

### Prerequisites

1. Docker and Docker Compose (for the database), or a local Postgres instance.
2. Go 1.25.3+
3. Node.js 25+ and pnpm 10+
4. A `.env` file with database credentials

**Note:** The test server automatically starts:
- Backend server on `http://localhost:8080`
- Test IMAP server on `localhost:1143`
- Test SMTP server on `localhost:1025`
- Seeds test data (sample emails)

If using Docker for the DB: ensure the database is running: `docker compose up -d db`

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

