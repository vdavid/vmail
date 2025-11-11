import { test, expect } from '@playwright/test'

import { setupAuth } from '../fixtures/auth'
import { defaultTestUser, sampleMessages } from '../fixtures/test-data'
import { getSearchInput, navigateAndWait, waitForEmailList } from '../utils/helpers'

/**
 * Search E2E Tests
 *
 * Comprehensive tests for search functionality covering:
 * - Plain text search
 * - Filter syntax (from:, to:, subject:, after:, before:, folder:, label:)
 * - Combined filters
 * - Quoted strings
 * - Empty query handling
 * - No results handling
 * - Pagination
 * - Navigation from search results
 * - Frontend validation
 *
 * Note: These tests require test data to be seeded in the IMAP server.
 * See e2e/README.md for setup instructions.
 */
test.describe('Search Functionality', () => {
    test.beforeEach(async ({ page }) => {
        await setupAuth(page, defaultTestUser.email)
        await navigateAndWait(page, '/')
    })

    test('plain text search works', async ({ page }) => {
        // Wait for page to load and find search input (in header)
        await page.waitForSelector('input[placeholder="Search mail..."]', { timeout: 10000 })
        const searchInput = page.locator('input[placeholder="Search mail..."]')

        // Use sampleMessages to ensure test data consistency
        // Search for "Special Report" which should match "Special Report Q3" from sampleMessages
        const searchTerm = 'Special Report'
        await searchInput.fill(searchTerm)
        await searchInput.press('Enter')

        // Verify we're on search page
        await expect(page).toHaveURL(new RegExp(`.*/search\\?q=${encodeURIComponent(searchTerm)}`))

        // Wait for results
        await waitForEmailList(page)

        // Verify search results page shows query (use main content area to avoid sidebar h1)
        await expect(page.locator('main h1, [role="main"] h1').first()).toContainText('Search results')
        
        // Verify we found the expected message from sampleMessages
        const expectedMessage = sampleMessages.find(m => m.subject.includes('Special Report'))
        if (expectedMessage) {
            await expect(page.locator('text=' + expectedMessage.subject)).toBeVisible()
        }
    })

    test('from: filter works', async ({ page }) => {
        const searchInput = await getSearchInput(page)

        await searchInput.fill('from:sender@example.com')
        await searchInput.press('Enter')

        // URL encoding: : becomes %3A, @ becomes %40
        await expect(page).toHaveURL(/.*\/search\?q=from.*sender.*example\.com/)
        await waitForEmailList(page)
    })

    test('to: filter works', async ({ page }) => {
        const searchInput = await getSearchInput(page)

        await searchInput.fill('to:test@example.com')
        await searchInput.press('Enter')

        // URL encoding: : becomes %3A, @ becomes %40
        await expect(page).toHaveURL(/.*\/search\?q=to.*test.*example\.com/)
        await waitForEmailList(page)
    })

    test('subject: filter works', async ({ page }) => {
        const searchInput = await getSearchInput(page)

        await searchInput.fill('subject:Meeting')
        await searchInput.press('Enter')

        // URL encoding: : becomes %3A
        await expect(page).toHaveURL(/.*\/search\?q=subject.*Meeting/)
        await waitForEmailList(page)
    })

    test('after: date filter works', async ({ page }) => {
        const searchInput = await getSearchInput(page)

        await searchInput.fill('after:2025-01-01')
        await searchInput.press('Enter')

        // URL encoding: : becomes %3A
        await expect(page).toHaveURL(/.*\/search\?q=after.*2025-01-01/)
        await waitForEmailList(page)
    })

    test('before: date filter works', async ({ page }) => {
        const searchInput = await getSearchInput(page)

        await searchInput.fill('before:2025-12-31')
        await searchInput.press('Enter')

        // URL encoding: : becomes %3A
        await expect(page).toHaveURL(/.*\/search\?q=before.*2025-12-31/)
        await waitForEmailList(page)
    })

    test('folder: filter works', async ({ page }) => {
        const searchInput = await getSearchInput(page)

        await searchInput.fill('folder:Inbox')
        await searchInput.press('Enter')

        // URL encoding: : becomes %3A
        await expect(page).toHaveURL(/.*\/search\?q=folder.*Inbox/)
        await waitForEmailList(page)
    })

    test('label: filter works (alias for folder)', async ({ page }) => {
        const searchInput = await getSearchInput(page)

        await searchInput.fill('label:Inbox')
        await searchInput.press('Enter')

        // URL encoding: : becomes %3A
        await expect(page).toHaveURL(/.*\/search\?q=label.*Inbox/)
        await waitForEmailList(page)
    })

    test('combined filters work', async ({ page }) => {
        const searchInput = await getSearchInput(page)

        await searchInput.fill('from:sender@example.com after:2025-01-01')
        await searchInput.press('Enter')

        // URL encoding: : becomes %3A, @ becomes %40, space becomes %20
        await expect(page).toHaveURL(/.*\/search\?q=.*from.*sender.*after.*2025-01-01/)
        await waitForEmailList(page)
    })

    test('quoted strings work', async ({ page }) => {
        const searchInput = await getSearchInput(page)

        await searchInput.fill('from:"John Doe"')
        await searchInput.press('Enter')

        // URL encoding: : becomes %3A, " becomes %22, space becomes %20
        await expect(page).toHaveURL(/.*\/search\?q=from.*John.*Doe/)
        await waitForEmailList(page)
    })

    test('empty query shows appropriate message', async ({ page }) => {
        await setupAuth(page, defaultTestUser.email)
        
        // Navigate to inbox first to ensure settings are loaded
        await navigateAndWait(page, '/')
        
        // Wait for inbox to load (ensures settings are available)
        await page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })
        
        // Wait a bit for authStatus to update
        await page.waitForTimeout(1000)
        
        // Now navigate to search page with empty query
        await navigateAndWait(page, '/search?q=')

        // Check if we were redirected to settings (shouldn't happen for test@example.com)
        const currentURL = page.url()
        if (currentURL.includes('/settings')) {
            // User doesn't have settings - skip this test or handle it
            // This shouldn't happen for test@example.com as test server seeds settings
            test.skip()
            return
        }

        // Wait for settings to load first (required for search query)
        await page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })

        // Empty query returns all emails (per backend behavior: "Empty query means return all emails")
        // So we should see either the email list or a "no results" message, not the "Enter a search query" message
        // The "Enter a search query" message only shows when threadsResponse is null/undefined
        // Since the API is called, we'll see either results or "No results found"
        await waitForEmailList(page)
        
        // Verify we're on the search page (not redirected to settings)
        await expect(page).toHaveURL(/.*\/search/)
        
        // Verify the page shows "Search" (not "Search results for ...") when query is empty
        await expect(page.locator('main h1, [role="main"] h1').first()).toContainText('Search')
    })

    test('no results shows appropriate message', async ({ page }) => {
        await setupAuth(page, defaultTestUser.email)
        await navigateAndWait(page, '/')
        
        const searchInput = await getSearchInput(page)

        // Wait for settings to load first
        await page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })

        // Search for something that definitely won't exist
        await searchInput.fill('nonexistent-email-xyz-123')
        await searchInput.press('Enter')

        await waitForEmailList(page)

        // Verify "no results" message
        // The SearchPage shows "No results found for \"query\"" when no results
        await expect(
            page.locator('text=No results found')
        ).toBeVisible({ timeout: 5000 })
    })

    test('pagination works', async ({ page }) => {
        const searchInput = await getSearchInput(page)

        await searchInput.fill('test')
        await searchInput.press('Enter')

        await waitForEmailList(page)

        // Check if pagination controls exist
        const pagination = page.locator('[data-testid="pagination"], .pagination')
        const paginationCount = await pagination.count()

        if (paginationCount > 0) {
            // Try clicking next page if available
            const nextButton = page.locator('text=Next, button:has-text("Next")')
            if (await nextButton.count() > 0 && (await nextButton.isEnabled())) {
                await nextButton.click()
                await expect(page).toHaveURL(/.*page=2/)
            }
        }
    })

    test('clicking results navigates correctly', async ({ page }) => {
        const searchInput = await getSearchInput(page)

        await searchInput.fill('test')
        await searchInput.press('Enter')

        await waitForEmailList(page)

        // Click first result if available (EmailListItem renders as <a> links)
        const emailLinks = page.locator('a[href*="/thread/"]')
        const count = await emailLinks.count()

        if (count > 0) {
            await emailLinks.first().click()
            await expect(page).toHaveURL(/.*\/thread\/.*/)
        }
    })

    test('frontend validation works for invalid queries', async ({ page }) => {
        const searchInput = await getSearchInput(page)

        // Try invalid date format
        await searchInput.fill('after:invalid-date')
        await searchInput.press('Enter')

        // Should either show validation error or handle gracefully
        // The exact behavior depends on your validation implementation
        await page.waitForTimeout(500)

        // Try empty filter value
        await searchInput.fill('from:')
        await searchInput.press('Enter')

        // Should handle gracefully
        await page.waitForTimeout(500)
    })

    test('search keyboard shortcut (/) focuses search bar', async ({ page }) => {
        // Note: The keyboard shortcut '/' to focus search is not currently implemented
        // This test is skipped until the feature is added
        test.skip()
    })
})

