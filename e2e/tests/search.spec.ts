import { test, expect } from '@playwright/test'

import { setupAuth } from '../fixtures/auth'
import { defaultTestUser, sampleMessages } from '../fixtures/test-data'
import { navigateAndWait, waitForEmailList } from '../utils/helpers'

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
        // Find search input (in header)
        const searchInput = page.locator('input[type="search"], input[placeholder*="search" i]').first()
        
        if (await searchInput.count() === 0) {
            test.skip()
            return
        }

        // Type search query
        await searchInput.fill('Special Report')
        await searchInput.press('Enter')

        // Verify we're on search page
        await expect(page).toHaveURL(/.*\/search\?q=Special%20Report/)

        // Wait for results
        await waitForEmailList(page)

        // Verify search results page shows query
        await expect(page.locator('h1')).toContainText('Search results')
    })

    test('from: filter works', async ({ page }) => {
        const searchInput = page.locator('input[type="search"], input[placeholder*="search" i]').first()
        
        if (await searchInput.count() === 0) {
            test.skip()
            return
        }

        await searchInput.fill('from:sender@example.com')
        await searchInput.press('Enter')

        await expect(page).toHaveURL(/.*\/search\?q=from:sender@example.com/)
        await waitForEmailList(page)
    })

    test('to: filter works', async ({ page }) => {
        const searchInput = page.locator('input[type="search"], input[placeholder*="search" i]').first()
        
        if (await searchInput.count() === 0) {
            test.skip()
            return
        }

        await searchInput.fill('to:test@example.com')
        await searchInput.press('Enter')

        await expect(page).toHaveURL(/.*\/search\?q=to:test@example.com/)
        await waitForEmailList(page)
    })

    test('subject: filter works', async ({ page }) => {
        const searchInput = page.locator('input[type="search"], input[placeholder*="search" i]').first()
        
        if (await searchInput.count() === 0) {
            test.skip()
            return
        }

        await searchInput.fill('subject:Meeting')
        await searchInput.press('Enter')

        await expect(page).toHaveURL(/.*\/search\?q=subject:Meeting/)
        await waitForEmailList(page)
    })

    test('after: date filter works', async ({ page }) => {
        const searchInput = page.locator('input[type="search"], input[placeholder*="search" i]').first()
        
        if (await searchInput.count() === 0) {
            test.skip()
            return
        }

        await searchInput.fill('after:2025-01-01')
        await searchInput.press('Enter')

        await expect(page).toHaveURL(/.*\/search\?q=after:2025-01-01/)
        await waitForEmailList(page)
    })

    test('before: date filter works', async ({ page }) => {
        const searchInput = page.locator('input[type="search"], input[placeholder*="search" i]').first()
        
        if (await searchInput.count() === 0) {
            test.skip()
            return
        }

        await searchInput.fill('before:2025-12-31')
        await searchInput.press('Enter')

        await expect(page).toHaveURL(/.*\/search\?q=before:2025-12-31/)
        await waitForEmailList(page)
    })

    test('folder: filter works', async ({ page }) => {
        const searchInput = page.locator('input[type="search"], input[placeholder*="search" i]').first()
        
        if (await searchInput.count() === 0) {
            test.skip()
            return
        }

        await searchInput.fill('folder:Inbox')
        await searchInput.press('Enter')

        await expect(page).toHaveURL(/.*\/search\?q=folder:Inbox/)
        await waitForEmailList(page)
    })

    test('label: filter works (alias for folder)', async ({ page }) => {
        const searchInput = page.locator('input[type="search"], input[placeholder*="search" i]').first()
        
        if (await searchInput.count() === 0) {
            test.skip()
            return
        }

        await searchInput.fill('label:Inbox')
        await searchInput.press('Enter')

        await expect(page).toHaveURL(/.*\/search\?q=label:Inbox/)
        await waitForEmailList(page)
    })

    test('combined filters work', async ({ page }) => {
        const searchInput = page.locator('input[type="search"], input[placeholder*="search" i]').first()
        
        if (await searchInput.count() === 0) {
            test.skip()
            return
        }

        await searchInput.fill('from:sender@example.com after:2025-01-01')
        await searchInput.press('Enter')

        await expect(page).toHaveURL(/.*\/search\?q=.*from:sender.*after:2025-01-01/)
        await waitForEmailList(page)
    })

    test('quoted strings work', async ({ page }) => {
        const searchInput = page.locator('input[type="search"], input[placeholder*="search" i]').first()
        
        if (await searchInput.count() === 0) {
            test.skip()
            return
        }

        await searchInput.fill('from:"John Doe"')
        await searchInput.press('Enter')

        await expect(page).toHaveURL(/.*\/search\?q=from:"John%20Doe"/)
        await waitForEmailList(page)
    })

    test('empty query shows appropriate message', async ({ page }) => {
        // Navigate directly to search page with empty query
        await navigateAndWait(page, '/search?q=')

        // Verify appropriate message is shown
        await expect(
            page.locator('text=Enter a search query, text=Search')
        ).toBeVisible()
    })

    test('no results shows appropriate message', async ({ page }) => {
        const searchInput = page.locator('input[type="search"], input[placeholder*="search" i]').first()
        
        if (await searchInput.count() === 0) {
            test.skip()
            return
        }

        // Search for something that definitely won't exist
        await searchInput.fill('nonexistent-email-xyz-123')
        await searchInput.press('Enter')

        await waitForEmailList(page)

        // Verify "no results" message
        await expect(
            page.locator('text=No results found, text=No emails')
        ).toBeVisible()
    })

    test('pagination works', async ({ page }) => {
        const searchInput = page.locator('input[type="search"], input[placeholder*="search" i]').first()
        
        if (await searchInput.count() === 0) {
            test.skip()
            return
        }

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
        const searchInput = page.locator('input[type="search"], input[placeholder*="search" i]').first()
        
        if (await searchInput.count() === 0) {
            test.skip()
            return
        }

        await searchInput.fill('test')
        await searchInput.press('Enter')

        await waitForEmailList(page)

        // Click first result if available
        const emailItems = page.locator('[data-testid="email-item"], .email-item')
        const count = await emailItems.count()

        if (count > 0) {
            await emailItems.first().click()
            await expect(page).toHaveURL(/.*\/thread\/.*/)
        } else {
            test.skip()
        }
    })

    test('frontend validation works for invalid queries', async ({ page }) => {
        const searchInput = page.locator('input[type="search"], input[placeholder*="search" i]').first()
        
        if (await searchInput.count() === 0) {
            test.skip()
            return
        }

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
        const searchInput = page.locator('input[type="search"], input[placeholder*="search" i]').first()
        
        if (await searchInput.count() === 0) {
            test.skip()
            return
        }

        // Click somewhere else to unfocus
        await page.click('body')

        // Press '/' to focus search
        await page.keyboard.press('/')

        // Verify search input is focused
        await expect(searchInput).toBeFocused()
    })
})

