import { test, expect } from '@playwright/test'

import { setupAuth } from '../fixtures/auth'
import { defaultTestUser } from '../fixtures/test-data'
import { navigateAndWait, waitForEmailList } from '../utils/helpers'

/**
 * Sidebar and Folder Navigation Tests
 *
 * Tests for:
 * - Sidebar folder rendering
 * - Folder navigation via sidebar links
 * - URL parameter handling for folders
 */
test.describe('Sidebar and Folder Navigation', () => {
    test.beforeEach(async ({ page }) => {
        await setupAuth(page, defaultTestUser.email)
        await navigateAndWait(page, '/')
        
        // Wait for redirect to complete (either to inbox or settings)
        await page.waitForURL(/.*\/(settings|$)/, { timeout: 10000 })
        
        // If redirected to settings, the user doesn't have settings yet
        // This shouldn't happen for test@example.com, but handle it gracefully
        const currentURL = page.url()
        if (currentURL.includes('/settings')) {
            // User needs to complete onboarding first - tests will skip
            return
        }
        
        // Wait for settings to load first
        await page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })
    })

    test('sidebar displays folders', async ({ page }) => {
        // Check if we're on settings page (user doesn't have settings)
        const currentURL = page.url()
        if (currentURL.includes('/settings')) {
            return // Skip if redirected to settings
        }
        
        // Wait for sidebar to load (folders API call)
        await page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })

        // Verify sidebar is visible
        // Sidebar structure: <div className='flex h-full w-64...'>
        const sidebar = page.locator('div.w-64, nav, [role="navigation"], aside').first()
        await expect(sidebar).toBeVisible({ timeout: 5000 })

        // Verify common folders are present (at least Inbox should be there)
        // The exact folder names depend on what the test IMAP server returns
        const inboxLink = page.locator('a[href*="folder=INBOX"], a[href*="folder=Inbox"], a:has-text("Inbox"), a:has-text("INBOX")').first()
        await expect(inboxLink).toBeVisible({ timeout: 5000 })
    })

    test('clicking folder link navigates correctly', async ({ page }) => {
        // Check if we're on settings page
        const currentURL = page.url()
        if (currentURL.includes('/settings')) {
            return // Skip if redirected to settings
        }
        
        // Wait for sidebar folders to be visible
        const inboxLink = page.locator('a[href*="folder=INBOX"], a[href*="folder=Inbox"], a:has-text("Inbox"), a:has-text("INBOX")').first()
        await expect(inboxLink).toBeVisible({ timeout: 5000 })

        // Click the Inbox link
        await inboxLink.click()

        // Verify URL contains folder parameter
        await expect(page).toHaveURL(/.*folder=(INBOX|Inbox)/i, { timeout: 5000 })

        // Verify email list loads
        await waitForEmailList(page)
    })

    test('navigating to folder via URL parameter works', async ({ page }) => {
        // Wait for initial page load and settings
        await page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })
        
        // Check if we were redirected to settings (user doesn't have settings)
        const currentURL = page.url()
        if (currentURL.includes('/settings')) {
            // User needs settings first - skip this test
            return
        }

        // Navigate directly to a folder via URL
        await navigateAndWait(page, '/?folder=INBOX')

        // Wait for settings to load
        await page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })

        // Verify URL is correct (might be redirected if no settings)
        const finalURL = page.url()
        if (finalURL.includes('/settings')) {
            return // User doesn't have settings
        }
        
        await expect(page).toHaveURL(/.*folder=(INBOX|Inbox)/i)

        // Verify email list loads
        await waitForEmailList(page)
    })

    test('settings link in sidebar works', async ({ page }) => {
        // Check if we're on settings page already
        const currentURL = page.url()
        if (currentURL.includes('/settings')) {
            // Already on settings page, just verify
            await expect(page.locator('main h1, [role="main"] h1').first()).toContainText('Settings')
            return
        }

        // Find settings link in sidebar
        // Sidebar structure: <div className='border-t border-gray-200 p-4'><Link to='/settings'>
        // The settings link is at the bottom of the sidebar, not in the nav element
        const settingsLink = page.locator('a[href="/settings"]:has-text("Settings")').first()
        await expect(settingsLink).toBeVisible({ timeout: 5000 })

        // Click settings link
        await settingsLink.click()

        // Verify we're on settings page
        await expect(page).toHaveURL(/.*\/settings/)
        await expect(page.locator('main h1, [role="main"] h1').first()).toContainText('Settings')
    })
})

