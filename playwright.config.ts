import { defineConfig, devices } from '@playwright/test'

/**
 * See https://playwright.dev/docs/test-configuration.
 */
export default defineConfig({
    testDir: './e2e/tests',
    fullyParallel: true,
    forbidOnly: !!process.env.CI,
    retries: process.env.CI ? 2 : 0,
    workers: process.env.CI ? 1 : undefined,
    reporter: 'html',
    // Use a separate TypeScript config to avoid Vitest conflicts
    // This ensures Playwright doesn't load Vitest's setup files
    use: {
        baseURL: 'http://localhost:8080',
        trace: 'on-first-retry',
    },

    projects: [
        {
            name: 'chromium',
            use: { ...devices['Desktop Chrome'] },
        },
    ],

    // Run the test server before starting the tests
    // The test server starts backend + test IMAP/SMTP servers
    webServer: {
        command: 'cd backend && go run ./cmd/test-server',
        url: 'http://localhost:8080',
        reuseExistingServer: !process.env.CI,
        timeout: 120 * 1000,
        env: {
            VMAIL_TEST_MODE: 'true',
        },
    },
})

