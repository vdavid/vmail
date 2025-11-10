#!/usr/bin/env node
// Script to run Playwright without Vitest interference
// Uses a clean Node.js environment to prevent Vitest from loading

import { spawn } from 'child_process'
import { fileURLToPath } from 'url'
import { dirname, join } from 'path'

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)
const projectRoot = join(__dirname, '../..')
const frontendDir = join(projectRoot, 'frontend')

const args = process.argv.slice(2)
// Default to 'test' if no args provided, otherwise use provided args
// Always include the config path
const playwrightArgs = args.length > 0 
    ? [...args, '--config=../playwright.config.ts']
    : ['test', '--config=../playwright.config.ts']

// Create a clean environment without Vitest
const cleanEnv = { ...process.env }
// Remove Vitest-related environment variables
delete cleanEnv.VITEST
delete cleanEnv.VITEST_WORKER_ID
delete cleanEnv.VITEST_POOL_ID

// Use pnpm exec to run playwright in a clean environment
// This prevents Vitest's module loader from interfering
const playwright = spawn('pnpm', ['exec', 'playwright', ...playwrightArgs], {
    cwd: frontendDir,
    stdio: 'inherit',
    shell: false,
    env: {
        ...cleanEnv,
        PLAYWRIGHT: '1',
        NODE_OPTIONS: '--no-warnings',
    },
})

playwright.on('close', (code) => {
    process.exit(code || 0)
})

