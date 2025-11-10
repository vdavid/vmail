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
    await page.click('button[type="submit"]')
    
    // Wait for navigation away from the settings page (indicates success)
    await page.waitForURL('**/', { timeout: 5000 })
}

/**
 * Waits for the email list to load.
 */
export async function waitForEmailList(page: Page) {
    // Wait for either the email list or "no emails" message
    await Promise.race([
        page.waitForSelector('[data-testid="email-list"]', { timeout: 5000 }).catch(() => null),
        page.waitForSelector('text=No results found', { timeout: 5000 }).catch(() => null),
        page.waitForSelector('text=Enter a search query', { timeout: 5000 }).catch(() => null),
    ])
}

/**
 * Clicks on the first email in the list.
 */
export async function clickFirstEmail(page: Page) {
    await page.locator('[data-testid="email-item"]').first().click()
    await page.waitForURL('**/thread/**', { timeout: 5000 })
}

