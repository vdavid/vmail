# Testing guidelines

## Unit tests

### Frontend

We use Jest and [React Testing Library](https://testing-library.com/).

* Test all simple, reusable components (e.g., `Button`, `Checkbox`).
* Test all utility functions (e.g., date formatting, search query parsing).
* Test custom hooks in isolation (e.g., `useKeyboardShortcuts`).

### Backend

We use Go's `testing` package

* **`imap` package:** Mock the IMAP server connection.
    * Test `TestParseThreadResponse`: Feed it a sample `* (THREAD ...)` string and assert that it builds
      the correct Go struct tree.
    * Test `TestBuildSearchQuery`: Feed it `"from:george after:2025"` and assert it creates the correct IMAP
      `SEARCH` query string.
* **`api` package:** Use `httptest` to test handlers. Mock the `imap` and `db` services.
    * Test `TestGetThreadsHandler`: Send a mock request and ensure it calls the `imap` service and returns
      the correct JSON.

## Integration tests

### Frontend

We use the [React Testing Library](https://testing-library.com/).

* Test feature flows by mocking the API layer (`msw` or `jest.mock`).
* "Selecting three emails and pressing 'e' calls `api.archive` with the three correct IDs."
* "Typing in the composer and pausing triggers the `api.saveDraft` mock."
* "Loading the Inbox page displays a list of three emails returned from the `api.getThreads` mock."

### Backend

* Use `testcontainers-go` to spin up a **real Postgres DB** for tests.
* Test the full flow: `api handler -> db package -> test postgres DB`.
* Example: "Call the draft saving endpoint and then query the test DB to ensure the draft was written correctly."

## End-to-end tests

We use Playwright.

* Test the *full*, running application.
* "User can log in (mocking Authelia), see the inbox, click an email, click 'Reply', type 'Test', click 'Send',
  and then find that email in the 'Sent' folder."

### Running E2E tests

End-to-end (E2E) tests use Playwright. They test the integration between
frontend, backend, database, and test IMAP/SMTP servers.

**When to run E2E tests:**

- Before merging a PR
- Before a release
- When adding major features
- In CI (automated)

For daily development, unit and integration tests are usually enough.

**Prerequisites:**

- Go 1.25.3+
- Node.js 25+ and pnpm 10+
- A `.env` file with database credentials (see `.env.example`)

**Running E2E tests:**

```bash
cd frontend
pnpm test:e2e
```

The test server (`backend/cmd/test-server`) automatically:
- Starts a test IMAP server on `localhost:1143`
- Starts a test SMTP server on `localhost:1025`
- Starts the backend server on `localhost:11765` (E2E test port)
- Seeds test data (sample emails)
- Sets `VMAIL_TEST_MODE=true` for non-TLS connections

**Test server credentials:**

- IMAP: `username` / `password` on `localhost:1143`
- SMTP: `test-user` / `test-pass` on `localhost:1025`

These match the values in `e2e/fixtures/test-data.ts`.

**Troubleshooting:**

- If tests fail to connect, ensure ports 11765, 7557, 1143, and 1025 are not in use
- If the database connection fails, ensure Docker Compose is running: `docker compose up -d db`
- Check the test server logs for detailed error messages
