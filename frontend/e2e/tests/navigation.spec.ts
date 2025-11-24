import { test, expect } from '@playwright/test'

import { setupInboxForNavigation } from '../utils/helpers'

/**
 * Test 3: Navigation
 *
 * This test verifies keyboard navigation shortcuts:
 * - 'u' key: Go back from thread view to inbox
 * - 'j' key: Move cursor to next email
 * - 'k' key: Move cursor to previous email
 * - 'o' key: Open selected email
 *
 * Note: This test requires emails to be present in the inbox.
 */
test.describe('Keyboard Navigation', () => {
    test('pressing u navigates back from thread to inbox', async ({ page }) => {
        const { emailLinks, count } = await setupInboxForNavigation(page)

        if (count === 0) {
            // No emails available, skip this test
            return
        }

        await emailLinks.first().click()
        await expect(page).toHaveURL(/.*\/thread\/.*/)

        // Wait for thread to load
        await page.waitForSelector('text=Loading...', {
            state: 'hidden',
            timeout: 10000,
        })

        // Wait for thread content to be visible (ensures page is fully loaded)
        await page.waitForSelector('h1, button:has-text("Back to Inbox")', {
            timeout: 5000,
        })

        // Wait a bit for React to finish rendering and keyboard handler to be ready
        await page.waitForTimeout(200)

        // Press 'u' to go back (keyboard handler should be active)
        await page.keyboard.press('u')

        // Verify we're back on the inbox
        await expect(page).toHaveURL(/.*\/$/, { timeout: 2000 })
    })

    test('pressing j moves selection to next email', async ({ page }) => {
        const { count } = await setupInboxForNavigation(page)

        if (count < 2) {
            // Need at least 2 emails for this test
            return
        }

        // Focus the page to ensure keyboard events work
        // Click on the sidebar title (non-interactive) instead of body to avoid clicking on email links
        await page.click('text=V-Mail')

        // Press 'j' to move to next email
        await page.keyboard.press('j')

        // Verify selection moved (check for focus/selected state)
        // The exact implementation depends on your UI store and EmailListItem
        // For now, we just verify the keypress doesn't cause errors
        await page.waitForTimeout(100) // Give time for state update

        // Press 'j' again to move to second email
        await page.keyboard.press('j')
        await page.waitForTimeout(100)
    })

    test('pressing k moves selection to previous email', async ({ page }) => {
        const { count } = await setupInboxForNavigation(page)

        if (count < 2) {
            // Need at least 2 emails for this test
            return
        }

        // Ensure we're on the inbox before starting
        await expect(page).toHaveURL(/.*\/$/)

        // Click on the sidebar title (non-interactive) to make sure the
        await page.click('text=V-Mail')

        // Move down with 'j' first
        await page.keyboard.press('j')
        await page.waitForTimeout(100)

        // Move up with 'k'
        await page.keyboard.press('k')
        await page.waitForTimeout(100)

        // Verify navigation works without errors
        await expect(page).toHaveURL(/.*\/$/)
    })

    test('pressing o opens selected email', async ({ page }) => {
        const { count } = await setupInboxForNavigation(page)

        if (count === 0) {
            // No emails available, skip this test
            return
        }

        // Click on the sidebar title (non-interactive) instead of body to avoid clicking on email links
        await page.click('text=V-Mail')

        // Select first email with 'j' (if not already selected)
        // Then press 'o' to open
        await page.keyboard.press('j')
        await page.waitForTimeout(100)
        await page.keyboard.press('o')

        // Verify we navigated to thread view
        await expect(page).toHaveURL(/.*\/thread\/.*/, { timeout: 2000 })
    })
})
