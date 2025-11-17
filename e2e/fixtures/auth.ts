import { Page } from '@playwright/test'

/**
 * Sets up authentication for E2E tests.
 * In test mode, the backend ValidateToken supports "email:user@example.com" format.
 * We intercept API requests and modify the Authorization header to include the email.
 */
export async function setupAuth(page: Page, userEmail: string = 'test@example.com') {
    // Intercept all API requests and test endpoints, and modify the Authorization header
    // to include the email in the token format "email:user@example.com"
    // This allows the backend to extract the email in test mode
    const addAuthHeader = async (route: any) => {
        const request = route.request()
        const headers = { ...request.headers() }
        
        // Always set/modify the Authorization header to include the email
        // Frontend sends "Bearer token" by default, we replace it with "email:user@example.com"
        headers['authorization'] = `Bearer email:${userEmail}`
        
        // Continue with the modified request
        await route.continue({ headers })
    }
    
    // Intercept API routes
    await page.route('**/api/**', addAuthHeader)
    // Intercept test routes
    await page.route('**/test/**', addAuthHeader)
}

/**
 * Mocks Authelia authentication endpoints if needed.
 * Currently not needed since backend has stub validation.
 */
export async function mockAuthelia(page: Page) {
    // Future: Mock Authelia token validation endpoint
    // await page.route('**/authelia/api/verify', route => {
    //     route.fulfill({ json: { email: 'test@example.com' } })
    // })
}

