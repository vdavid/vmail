import { test, expect } from '@playwright/test'

import { setupAuth } from '../fixtures/auth'
import { defaultTestUser } from '../fixtures/test-data'
import {
    fillSettingsForm,
    navigateAndWait,
    submitSettingsForm,
    waitForAppReady,
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
        // Setup auth (currently a no-op since backend accepts any token)
        await setupAuth(page, defaultTestUser.email)

        // Navigate to root - should redirect to /settings for new user
        await navigateAndWait(page, '/')

        // Verify we're on the settings page
        await expect(page).toHaveURL(/.*\/settings/)
        await expect(page.locator('h1')).toContainText('Settings')

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
        await waitForAppReady(page)

        // Verify we're no longer on the settings page
        await expect(page.locator('h1')).not.toContainText('Settings')
    })

    test('shows validation errors for empty required fields', async ({ page }) => {
        await setupAuth(page, defaultTestUser.email)
        await navigateAndWait(page, '/settings')

        // Try to submit without filling required fields
        await page.click('button[type="submit"]')

        // HTML5 validation should prevent submission
        // Check that required fields are marked as invalid
        const imapServerInput = page.locator('input[name="imap_server_hostname"]')
        await expect(imapServerInput).toBeInvalid()
    })
})

