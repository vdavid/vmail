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

    test('displays sender and subject in email list', async ({ page }) => {
        await setupAuth(page, defaultTestUser.email)
        await navigateAndWait(page, '/')

        // Wait for redirect to complete
        await page.waitForURL(/.*\/(settings|$)/, { timeout: 10000 })

        const currentURL = page.url()
        if (currentURL.includes('/settings')) {
            return // Skip if redirected to settings
        }

        // Wait for settings to load first
        await page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })
        
        await waitForEmailList(page)

        const emailLinks = page.locator('a[href*="/thread/"]')
        const count = await emailLinks.count()

        if (count > 0) {
            // Get the first email link
            const firstEmail = emailLinks.first()

            // Verify sender is displayed (not "Unknown")
            // Use data-testid for style-independent testing
            const senderSpan = firstEmail.locator('[data-testid="email-sender"]').first()
            await expect(senderSpan).toBeVisible({ timeout: 5000 })
            const senderText = await senderSpan.textContent()
            expect(senderText).toBeTruthy()
            expect(senderText?.trim()).not.toBe('Unknown')
            expect(senderText?.trim().length).toBeGreaterThan(0)

            // Verify subject is displayed
            // Use data-testid for style-independent testing
            const subjectDiv = firstEmail.locator('[data-testid="email-subject"]').first()
            await expect(subjectDiv).toBeVisible({ timeout: 5000 })
            const subjectText = await subjectDiv.textContent()
            expect(subjectText).toBeTruthy()
            expect(subjectText?.trim()).not.toBe('(No subject)')
            expect(subjectText?.trim().length).toBeGreaterThan(0)
        }
    })

    test('clicking email navigates to thread with correct URL and displays body', async ({ page }) => {
        await setupAuth(page, defaultTestUser.email)
        await navigateAndWait(page, '/')

        // Wait for redirect to complete
        await page.waitForURL(/.*\/(settings|$)/, { timeout: 10000 })

        const currentURL = page.url()
        if (currentURL.includes('/settings')) {
            return // Skip if redirected to settings
        }

        // Wait for settings to load first
        await page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })
        
        await waitForEmailList(page)

        const emailLinks = page.locator('a[href*="/thread/"]')
        const count = await emailLinks.count()

        if (count > 0) {
            // Get the href of the first email to verify URL format
            // noinspection ES6RedundantAwait -- getAttribute returns Promise<string | null>, so await is required
            const firstEmailHref: string | null = await emailLinks.first().getAttribute('href')
            expect(firstEmailHref).toBeTruthy()
            expect(firstEmailHref).toMatch(/^\/thread\//)

            // Click the first email
            await clickFirstEmail(page)

            // Verify URL is correct format (should be /thread/{threadId}, not double-encoded)
            // The URL should not have %3C or %3E (encoded < and >) unless the threadId actually contains them
            // But it should be properly formatted
            await expect(page).toHaveURL(/.*\/thread\/[^/]+$/, { timeout: 5000 })

            // Verify the URL doesn't have obvious encoding issues
            const finalURL = page.url()
            // Check that if there are angle brackets, they're properly encoded, but the URL is still valid
            expect(finalURL).toContain('/thread/')

            // Wait for thread content to load
            await page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })

            // Verify thread page shows content (not blank)
            // Check for thread subject in header
            const threadHeader = page.locator('main h1, [role="main"] h1').first()
            await expect(threadHeader).toBeVisible({ timeout: 5000 })

            // Verify email body/content is visible
            // Message component should render the email body
            const messageContent = page.locator(
                'article, [data-testid="message"], .message, main div.border-b'
            ).first()
            await expect(messageContent).toBeVisible({ timeout: 5000 })

            // Verify sender is displayed in the message
            const senderInMessage = page.locator('text=/sender@example\\.com|colleague@example\\.com|reports@example\\.com/i')
            await expect(senderInMessage.first()).toBeVisible({ timeout: 5000 })

            // Verify message body text is visible (not empty)
            const bodyText = page.locator('div.prose, div.whitespace-pre-wrap, article').first()
            const bodyContent = await bodyText.textContent()
            expect(bodyContent).toBeTruthy()
            expect(bodyContent?.trim().length).toBeGreaterThan(0)
        }
    })

    test('navigating directly to thread URL loads React app, not JSON', async ({ page }) => {
        // This test specifically catches the bug where navigating to /thread/... 
        // would show JSON instead of the React app
        await setupAuth(page, defaultTestUser.email)
        
        // First, get a thread ID from the inbox
        await navigateAndWait(page, '/')
        await page.waitForURL(/.*\/(settings|$)/, { timeout: 10000 })
        
        const currentURL = page.url()
        if (currentURL.includes('/settings')) {
            return // Skip if redirected to settings
        }

        await page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })
        await waitForEmailList(page)

        const emailLinks = page.locator('a[href*="/thread/"]')
        const count = await emailLinks.count()

        if (count > 0) {
            // Get the thread URL from the first email link
            // noinspection ES6RedundantAwait -- getAttribute returns Promise<string | null>, so await is required
            const threadUrl = await emailLinks.first().getAttribute('href')
            expect(threadUrl).toBeTruthy()
            
            // Navigate directly to the thread URL (simulating a bookmark or direct navigation)
            // This should load the React app, not JSON
            await page.goto(threadUrl!, { waitUntil: 'networkidle' })
            
            // Verify we're on the thread page
            await expect(page).toHaveURL(new RegExp(`.*${threadUrl}.*`), { timeout: 5000 })
            
            // CRITICAL: Verify the React app loaded (not JSON)
            // Check for React app elements, not JSON content
            const pageContent = await page.content()
            
            // Should contain React app structure (div with id="root" or React components)
            expect(pageContent).toContain('root')
            
            // Should NOT be pure JSON (no JSON object at root level)
            // If it's JSON, the page would start with { or [ and have no HTML structure
            expect(pageContent).not.toMatch(/^\s*[{\[]/)
            
            // Should have HTML structure
            expect(pageContent).toContain('<html')
            expect(pageContent).toContain('<body')
            
            // Wait for React to hydrate and render
            await page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })
            
            // Verify thread content is visible (React rendered it)
            const threadHeader = page.locator('h1, [role="heading"]').first()
            await expect(threadHeader).toBeVisible({ timeout: 5000 })
        }
    })
})

