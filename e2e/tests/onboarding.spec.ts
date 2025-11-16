import { test, expect } from '@playwright/test'

import { setupAuth } from '../fixtures/auth'
import { defaultTestUser } from '../fixtures/test-data'
import {
    fillSettingsForm,
    navigateAndWait,
    submitSettingsForm,
} from '../utils/helpers'

/**
 * Test 1: New User Onboarding Flow
 *
 * This test verifies that a new user (not yet set up) is redirected to
 * the settings page, can fill in their IMAP/SMTP credentials, and is
 * then redirected to the inbox.
 *
 * Note: This test currently requires manual server setup. See e2e/README.md
 */
test.describe('New User Onboarding', () => {
    test('redirects to settings and completes onboarding', async ({ page }) => {
        // Use a different email for new user test (test server creates settings for test@example.com)
        const newUserEmail = 'newuser@example.com'
        await setupAuth(page, newUserEmail)

        // Navigate to root - should redirect to /settings for new user
        await navigateAndWait(page, '/')

        // Wait for redirect to settings page
        await page.waitForURL(/.*\/settings/, { timeout: 10000 })
        
        // Wait for settings page to load (it loads asynchronously)
        await page.waitForSelector('h1:has-text("Settings")', { timeout: 10000 })

        // Verify we're on the settings page
        await expect(page).toHaveURL(/.*\/settings/)
        // Use main content area to avoid sidebar h1
        await expect(page.locator('main h1, [role="main"] h1').first()).toContainText('Settings')

        // Wait for form to be ready
        await page.waitForSelector('input[name="imap_server_hostname"]', { timeout: 10000 })

        // Fill in IMAP settings
        // Note: These values need to match your test IMAP server
        await fillSettingsForm(
            page,
            defaultTestUser.imapServer,
            defaultTestUser.imapUsername,
            defaultTestUser.imapPassword,
            defaultTestUser.smtpServer,
            defaultTestUser.smtpUsername,
            defaultTestUser.smtpPassword
        )

        // Submit the form
        await submitSettingsForm(page)

        // Verify we're redirected to the inbox
        await expect(page).toHaveURL(/.*\/$/)
        
        // Wait for inbox to load
        await page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })

        // Verify we're no longer on the settings page
        // Use main content area to avoid sidebar h1
        await expect(page.locator('main h1, [role="main"] h1').first()).not.toContainText('Settings')
    })

    test('shows validation errors for empty required fields', async ({ page }) => {
        // Use a different email for new user test
        const newUserEmail = 'newuser2@example.com'
        await setupAuth(page, newUserEmail)
        await navigateAndWait(page, '/settings')

        // Wait for form to be ready
        await page.waitForSelector('input[name="imap_server_hostname"]', { timeout: 10000 })

        // Try to submit without filling required fields
        // First, ensure the form is empty (clear any default values)
        await page.fill('input[name="imap_server_hostname"]', '')
        await page.fill('input[name="imap_username"]', '')
        await page.fill('input[name="smtp_server_hostname"]', '')
        await page.fill('input[name="smtp_username"]', '')

        // Try to submit
        const submitButton = page.locator('button[type="submit"]')
        await submitButton.click()

        // HTML5 validation should prevent submission
        // Check that required fields are marked as invalid
        // Note: HTML5 validation might not set :invalid immediately, so we check the form validity
        const imapServerInput = page.locator('input[name="imap_server_hostname"]')
        
        // Wait a bit for validation to trigger
        await page.waitForTimeout(100)
        
        // Check if the input is invalid (HTML5 validation)
        const isInvalid = await imapServerInput.evaluate((el: HTMLInputElement) => !el.validity.valid)
        expect(isInvalid).toBeTruthy()
    })
})

