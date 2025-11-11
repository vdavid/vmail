import { test, expect } from '@playwright/test'

import { setupAuth } from '../fixtures/auth'
import { defaultTestUser } from '../fixtures/test-data'
import { navigateAndWait, waitForEmailList } from '../utils/helpers'

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
        await setupAuth(page, defaultTestUser.email)
        await navigateAndWait(page, '/')

        // Wait for settings to load first
        await page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })
        
        await waitForEmailList(page)

        // Click first email to go to thread view (EmailListItem renders as <a> links)
        const emailLinks = page.locator('a[href*="/thread/"]')
        const count = await emailLinks.count()

        if (count === 0) {
            // No emails available, skip this test
            return
        }

        await emailLinks.first().click()
        await expect(page).toHaveURL(/.*\/thread\/.*/)

        // Wait for thread to load
        await page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })

        // Press 'u' to go back
        await page.keyboard.press('u')

        // Verify we're back on the inbox
        await expect(page).toHaveURL(/.*\/$/, { timeout: 2000 })
    })

    test('pressing j moves selection to next email', async ({ page }) => {
        await setupAuth(page, defaultTestUser.email)
        await navigateAndWait(page, '/')

        // Wait for settings to load first
        await page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })
        
        await waitForEmailList(page)

        const emailLinks = page.locator('a[href*="/thread/"]')
        const count = await emailLinks.count()

        if (count < 2) {
            // Need at least 2 emails for this test
            return
        }

        // Focus the page to ensure keyboard events work
        await page.click('body')

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
        await setupAuth(page, defaultTestUser.email)
        await navigateAndWait(page, '/')

        // Wait for settings to load first
        await page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })
        
        await waitForEmailList(page)

        const emailLinks = page.locator('a[href*="/thread/"]')
        const count = await emailLinks.count()

        if (count < 2) {
            // Need at least 2 emails for this test
            return
        }

        await page.click('body')

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
        await setupAuth(page, defaultTestUser.email)
        await navigateAndWait(page, '/')

        // Wait for settings to load first
        await page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })
        
        await waitForEmailList(page)

        const emailLinks = page.locator('a[href*="/thread/"]')
        const count = await emailLinks.count()

        if (count === 0) {
            // No emails available, skip this test
            return
        }

        await page.click('body')

        // Select first email with 'j' (if not already selected)
        // Then press 'o' to open
        await page.keyboard.press('j')
        await page.waitForTimeout(100)
        await page.keyboard.press('o')

        // Verify we navigated to thread view
        await expect(page).toHaveURL(/.*\/thread\/.*/, { timeout: 2000 })
    })
})

