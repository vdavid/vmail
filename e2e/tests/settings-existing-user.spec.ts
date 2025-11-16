import { test, expect } from '@playwright/test'

import { setupAuth } from '../fixtures/auth'
import { defaultTestUser } from '../fixtures/test-data'
import { navigateAndWait } from '../utils/helpers'

/**
 * Settings Page Tests for Existing Users
 *
 * Tests for:
 * - Loading existing settings
 * - Editing and saving settings
 * - Settings persistence
 */
test.describe('Settings Page (Existing User)', () => {
    test('loads existing settings', async ({ page }) => {
        await setupAuth(page, defaultTestUser.email)
        await navigateAndWait(page, '/settings')

        // Wait for settings to load
        await page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })

        // Verify we're on settings page
        await expect(page).toHaveURL(/.*\/settings/)
        await expect(page.locator('main h1, [role="main"] h1').first()).toContainText('Settings')

        // Verify form fields are populated with existing values
        // The test server seeds settings, so these should have values
        // Note: Test server uses 127.0.0.1 instead of localhost
        const imapServerInput = page.locator('input[name="imap_server_hostname"]')
        await expect(imapServerInput).toHaveValue(/127\.0\.0\.1:1143|localhost:1143/, { timeout: 5000 })

        const imapUsernameInput = page.locator('input[name="imap_username"]')
        await expect(imapUsernameInput).toHaveValue(/username/, { timeout: 5000 })
    })

    test('allows editing and saving settings', async ({ page }) => {
        await setupAuth(page, defaultTestUser.email)
        await navigateAndWait(page, '/settings')

        // Wait for settings to load
        await page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })

        // Wait for form to be ready
        await page.waitForSelector('input[name="imap_server_hostname"]', { timeout: 10000 })

        // Modify a setting (change IMAP server)
        const imapServerInput = page.locator('input[name="imap_server_hostname"]')
        await imapServerInput.clear()
        await imapServerInput.fill('imap.example.com:993')

        // Save settings (existing users don't redirect, they stay on settings page)
        const submitButton = page.locator('button[type="submit"]')
        await submitButton.click()

        // Wait for success message (existing users stay on settings page)
        await page.waitForSelector('text=Settings saved successfully', { timeout: 5000 })
        
        // Verify we're still on settings page
        await expect(page).toHaveURL(/.*\/settings/)
    })

    test('shows validation errors for invalid input', async ({ page }) => {
        await setupAuth(page, defaultTestUser.email)
        await navigateAndWait(page, '/settings')

        // Wait for settings to load
        await page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })
        await page.waitForSelector('input[name="imap_server_hostname"]', { timeout: 10000 })

        // Clear required fields
        await page.fill('input[name="imap_server_hostname"]', '')
        await page.fill('input[name="imap_username"]', '')
        await page.fill('input[name="smtp_server_hostname"]', '')
        await page.fill('input[name="smtp_username"]', '')

        // Try to submit
        const submitButton = page.locator('button[type="submit"]')
        await submitButton.click()

        // HTML5 validation should prevent submission
        // Check that required fields are marked as invalid
        const imapServerInput = page.locator('input[name="imap_server_hostname"]')
        await page.waitForTimeout(100)
        const isInvalid = await imapServerInput.evaluate((el: HTMLInputElement) => !el.validity.valid)
        expect(isInvalid).toBeTruthy()
    })
})

