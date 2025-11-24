import { defineConfig, devices } from '@playwright/test'

/**
 * See https://playwright.dev/docs/test-configuration.
 */
export default defineConfig({
    testDir: './frontend/e2e/tests',
    fullyParallel: true,
    forbidOnly: !!process.env.CI,
    retries: process.env.CI ? 2 : 0,
    workers: process.env.CI ? 1 : undefined,
    reporter: [['html', { open: 'never' }]],
    // Use a separate TypeScript config to avoid Vitest conflicts
    // This ensures Playwright doesn't load Vitest's setup files
    use: {
        baseURL: 'http://localhost:7557', // Frontend Vite dev server (E2E test port)
        trace: 'on-first-retry',
        screenshot: 'only-on-failure',
        video: 'retain-on-failure',
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
            url: 'http://localhost:11765',
            reuseExistingServer: false,
            timeout: 120 * 1000,
            env: {
                VMAIL_ENV: 'test',
                VMAIL_TEST_MODE: 'true',
                PORT: '11765', // Use different port for E2E tests
                VMAIL_IMAP_MAX_WORKERS: '50', // Increase max workers for faster tests
            },
        },
        {
            command: 'cd frontend && VITE_PORT=7557 VITE_API_URL=http://localhost:11765 pnpm dev',
            url: 'http://localhost:7557',
            reuseExistingServer: false,
            timeout: 60 * 1000,
        },
    ],
})

