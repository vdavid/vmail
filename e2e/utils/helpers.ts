import { Page } from '@playwright/test'

/**
 * Waits for the page to be fully loaded and ready.
 */
export async function waitForAppReady(page: Page) {
    // Wait for the main app to load
    await page.waitForSelector('body', { state: 'visible' })
    
    // Wait a bit for React to hydrate
    await page.waitForTimeout(500)
}

/**
 * Navigates to a URL and waits for the app to be ready.
 */
export async function navigateAndWait(page: Page, url: string) {
    await page.goto(url)
    await waitForAppReady(page)
}

/**
 * Fills in the settings form with test data.
 */
export async function fillSettingsForm(
    page: Page,
    imapServer: string,
    imapUsername: string,
    imapPassword: string,
    smtpServer: string,
    smtpUsername: string,
    smtpPassword: string
) {
    // Wait for form to be ready (settings page loads asynchronously)
    await page.waitForSelector('input[name="imap_server_hostname"]', { timeout: 10000 })
    
    // Fill IMAP settings
    await page.fill('input[name="imap_server_hostname"]', imapServer)
    await page.fill('input[name="imap_username"]', imapUsername)
    await page.fill('input[name="imap_password"]', imapPassword)
    
    // Fill SMTP settings
    await page.fill('input[name="smtp_server_hostname"]', smtpServer)
    await page.fill('input[name="smtp_username"]', smtpUsername)
    await page.fill('input[name="smtp_password"]', smtpPassword)
    
    // Use default values for other settings (they should have defaults)
}

/**
 * Submits the settings form.
 */
export async function submitSettingsForm(page: Page) {
    // Wait for submit button to be enabled
    await page.waitForSelector('button[type="submit"]:not([disabled])', { timeout: 5000 })
    await page.click('button[type="submit"]')
    
    // Wait for navigation away from the settings page (indicates success)
    // The form submission triggers a redirect to the inbox
    // Use a more flexible pattern that matches root path or inbox
    await page.waitForURL(/.*\/$/, { timeout: 10000 })
    
    // Wait for the page to finish loading
    await waitForAppReady(page)
}

/**
 * Waits for the email list to load.
 */
export async function waitForEmailList(page: Page) {
    // Wait for either the email list (EmailListItem components) or "no emails" message
    // EmailListItem renders as <a> links, so we wait for those or empty state messages
    await Promise.race([
        page.waitForSelector('a[href*="/thread/"]', { timeout: 10000 }).catch(() => null),
        page.waitForSelector('text=No threads found', { timeout: 10000 }).catch(() => null),
        page.waitForSelector('text=No results found', { timeout: 10000 }).catch(() => null),
        page.waitForSelector('text=Enter a search query', { timeout: 10000 }).catch(() => null),
        page.waitForSelector('text=Loading...', { timeout: 1000 }).then(() => 
            page.waitForSelector('text=Loading...', { state: 'hidden', timeout: 10000 })
        ).catch(() => null),
    ])
}

/**
 * Clicks on the first email in the list.
 */
export async function clickFirstEmail(page: Page) {
    // EmailListItem renders as <a> links with href="/thread/..."
    const firstEmailLink = page.locator('a[href*="/thread/"]').first()
    await firstEmailLink.waitFor({ state: 'visible', timeout: 5000 })
    await firstEmailLink.click()
    // Wait for navigation - URL should be properly formatted
    await page.waitForURL(/.*\/thread\/[^/]+/, { timeout: 5000 })
}

/**
 * Waits for and returns the search input from the header.
 */
export async function getSearchInput(page: Page) {
    await page.waitForSelector('input[placeholder="Search mail..."]', { timeout: 10000 })
    return page.locator('input[placeholder="Search mail..."]')
}

