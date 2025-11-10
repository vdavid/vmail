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
    reporter: [['html', { open: 'never' }]],
    // Use a separate TypeScript config to avoid Vitest conflicts
    // This ensures Playwright doesn't load Vitest's setup files
    use: {
        baseURL: 'http://localhost:7556', // Frontend Vite dev server
        trace: 'on-first-retry',
    },

    projects: [
        {
            name: 'chromium',
            use: { ...devices['Desktop Chrome'] },
        },
    ],

    // Run multiple web servers before starting the tests
    // 1. Backend API server (with test IMAP/SMTP servers)
    // 2. Frontend Vite dev server
    webServer: [
        {
            command: 'cd backend && go run ./cmd/test-server',
            url: 'http://localhost:8080',
            reuseExistingServer: false,
            timeout: 120 * 1000,
            env: {
                VMAIL_TEST_MODE: 'true',
            },
        },
        {
            command: 'cd frontend && pnpm dev',
            url: 'http://localhost:7556',
            reuseExistingServer: false,
            timeout: 60 * 1000,
        },
    ],
})

