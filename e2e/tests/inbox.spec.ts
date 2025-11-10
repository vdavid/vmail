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

        // Verify we're on the inbox (not redirected to settings)
        await expect(page).toHaveURL(/.*\/$/)

        // Wait for email list to load
        await waitForEmailList(page)

        // Verify sidebar shows folders
        // Note: This assumes the sidebar component exists
        // Adjust selector based on actual implementation
        const sidebar = page.locator('nav, [role="navigation"]').first()
        await expect(sidebar).toBeVisible()
    })

    test('displays thread list when emails exist', async ({ page }) => {
        await setupAuth(page, defaultTestUser.email)
        await navigateAndWait(page, '/')

        await waitForEmailList(page)

        // Check if email list is visible
        // The exact selector depends on your EmailListItem implementation
        // For now, we check for common patterns
        const emailList = page.locator('[data-testid="email-list"], .email-list, ul > li').first()
        
        // If emails exist, at least one should be visible
        // If no emails, we should see an empty state message
        const hasEmails = await emailList.count() > 0
        if (hasEmails) {
            await expect(emailList.first()).toBeVisible()
        } else {
            // Check for empty state
            await expect(
                page.locator('text=No emails, text=No results, text=Enter a search query')
            ).toBeVisible()
        }
    })

    test('navigates to thread view when clicking email', async ({ page }) => {
        await setupAuth(page, defaultTestUser.email)
        await navigateAndWait(page, '/')

        await waitForEmailList(page)

        // Try to click first email if it exists
        const emailItems = page.locator('[data-testid="email-item"], .email-item, ul > li')
        const count = await emailItems.count()

        if (count > 0) {
            await clickFirstEmail(page)

            // Verify we're on a thread page
            await expect(page).toHaveURL(/.*\/thread\/.*/)

            // Verify thread content is visible
            // Adjust selectors based on your Message component
            await expect(
                page.locator('article, [data-testid="message"], .message').first()
            ).toBeVisible()
        } else {
            // Skip test if no emails available
            test.skip()
        }
    })

    test('displays email body and attachments in thread view', async ({ page }) => {
        await setupAuth(page, defaultTestUser.email)
        await navigateAndWait(page, '/')

        await waitForEmailList(page)

        const emailItems = page.locator('[data-testid="email-item"], .email-item')
        const count = await emailItems.count()

        if (count > 0) {
            await clickFirstEmail(page)

            // Verify email body is visible
            // The exact selector depends on your Message component
            const messageBody = page
                .locator('[data-testid="message-body"], .message-body, article')
                .first()
            await expect(messageBody).toBeVisible()

            // Check for attachments if they exist
            // This is optional - only check if attachments are present
            const attachments = page.locator(
                '[data-testid="attachment"], .attachment, a[href*="attachment"]'
            )
            const attachmentCount = await attachments.count()
            if (attachmentCount > 0) {
                await expect(attachments.first()).toBeVisible()
            }
        } else {
            test.skip()
        }
    })
})

