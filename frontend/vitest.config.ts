import { resolve } from 'path'

import react from '@vitejs/plugin-react'
import { defineConfig } from 'vitest/config'

// Don't load Vitest config if Playwright is running
// This prevents Vitest globals from interfering with Playwright
const isPlaywright = process.env.PLAYWRIGHT === '1'

export default defineConfig({
    plugins: [react()],
    test: {
        // Only enable globals if not running Playwright
        globals: !isPlaywright,
        environment: isPlaywright ? undefined : 'jsdom',
        setupFiles: isPlaywright ? [] : ['./src/test/setup.ts'],
        css: true,
        // Exclude E2E tests from Vitest (they're for Playwright)
        exclude: [
            '**/node_modules/**',
            '**/dist/**',
            '**/e2e/**',
            '**/.{idea,git,cache,output,temp}/**',
        ],
    },
    resolve: {
        alias: {
            '@': resolve(__dirname, './src'),
        },
    },
})
