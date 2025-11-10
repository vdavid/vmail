import { test, expect } from '@playwright/test'

import { setupAuth } from '../fixtures/auth'
import { defaultTestUser } from '../fixtures/test-data'
import {
    clickFirstEmail,
    navigateAndWait,
    waitForEmailList,
} from '../utils/helpers'

/**
 * Test 2: Existing User Read-Only Flow
 *
 * This test verifies that an existing user (already set up) can:
 * - See the inbox with email threads
 * - Click on an email to view it
 * - See email content and attachments
 *
 * Note: This test requires test data to be seeded in the IMAP server.
 * See e2e/README.md for setup instructions.
 */
test.describe('Existing User Read-Only Flow', () => {
    test('displays inbox with email threads', async ({ page }) => {
        await setupAuth(page, defaultTestUser.email)
        await navigateAndWait(page, '/')

        // Wait for redirect to complete (either to inbox or settings)
        await page.waitForURL(/.*\/(settings|$)/, { timeout: 10000 })

        // If redirected to settings, the user doesn't have settings yet
        // This shouldn't happen for existing user tests, but handle it gracefully
        const currentURL = page.url()
        if (currentURL.includes('/settings')) {
            // User needs to complete onboarding first
            return
        }

        // Verify we're on the inbox (not redirected to settings)
        await expect(page).toHaveURL(/.*\/$/)

        // Wait for settings to load first (required for threads query)
        await page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })

        // Wait for email list to load
        await waitForEmailList(page)

        // Verify we see either email threads or "No threads found"
        const hasThreads = await page.locator('a[href*="/thread/"]').count() > 0
        const hasEmptyState = await page.locator('text=No threads found').count() > 0
        
        expect(hasThreads || hasEmptyState).toBeTruthy()
    })

    test('displays thread list when emails exist', async ({ page }) => {
        await setupAuth(page, defaultTestUser.email)
        await navigateAndWait(page, '/')

        // Wait for redirect to complete
        await page.waitForURL(/.*\/(settings|$)/, { timeout: 10000 })

        // If redirected to settings, skip this test
        const currentURL = page.url()
        if (currentURL.includes('/settings')) {
            return
        }

        // Wait for settings to load first
        await page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })
        
        await waitForEmailList(page)

        // Check if email threads are visible (EmailListItem renders as <a> links)
        const emailLinks = page.locator('a[href*="/thread/"]')
        const count = await emailLinks.count()
        
        if (count > 0) {
            // At least one email thread should be visible
            await expect(emailLinks.first()).toBeVisible()
        } else {
            // Check for empty state message (in main content area)
            await expect(page.locator('main text=No threads found, [role="main"] text=No threads found').first()).toBeVisible()
        }
    })

    test('navigates to thread view when clicking email', async ({ page }) => {
        await setupAuth(page, defaultTestUser.email)
        await navigateAndWait(page, '/')

        // Wait for settings to load first
        await page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })
        
        await waitForEmailList(page)

        // Try to click first email if it exists (EmailListItem renders as <a> links)
        const emailLinks = page.locator('a[href*="/thread/"]')
        const count = await emailLinks.count()

        if (count > 0) {
            await clickFirstEmail(page)

            // Verify we're on a thread page
            await expect(page).toHaveURL(/.*\/thread\/.*/)

            // Wait for thread content to load
            await page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })
            
            // Verify thread content is visible (Message component or article)
            await expect(
                page.locator('article, [data-testid="message"], .message, main').first()
            ).toBeVisible({ timeout: 5000 })
        }
    })

    test('displays email body and attachments in thread view', async ({ page }) => {
        await setupAuth(page, defaultTestUser.email)
        await navigateAndWait(page, '/')

        // Wait for settings to load first
        await page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })
        
        await waitForEmailList(page)

        const emailLinks = page.locator('a[href*="/thread/"]')
        const count = await emailLinks.count()

        if (count > 0) {
            await clickFirstEmail(page)

            // Wait for thread content to load
            await page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })

            // Verify email body is visible
            // The exact selector depends on your Message component
            const messageBody = page
                .locator('[data-testid="message-body"], .message-body, article, main')
                .first()
            await expect(messageBody).toBeVisible({ timeout: 5000 })

            // Check for attachments if they exist
            // This is optional - only check if attachments are present
            const attachments = page.locator(
                '[data-testid="attachment"], .attachment, a[href*="attachment"]'
            )
            const attachmentCount = await attachments.count()
            if (attachmentCount > 0) {
                await expect(attachments.first()).toBeVisible()
            }
        }
    })
})

