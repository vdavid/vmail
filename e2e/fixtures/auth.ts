import { Page } from '@playwright/test'

/**
 * Sets up authentication for E2E tests.
 * The backend currently accepts any Bearer token and returns "test@example.com".
 * This helper ensures the frontend sends the token in API requests.
 */
export async function setupAuth(page: Page, userEmail: string = 'test@example.com') {
    // The frontend currently uses a hardcoded 'Bearer token' in getAuthHeaders().
    // For E2E tests, we can rely on this or mock the API calls.
    // Since the backend ValidateToken is a stub, any token works.
    
    // For now, we don't need to do anything special since:
    // 1. Frontend sends 'Bearer token' by default
    // 2. Backend accepts any token and returns test@example.com
    
    // In the future, if we implement real auth, we'd:
    // - Mock Authelia endpoints
    // - Set auth tokens in localStorage/cookies
    // - Or use Playwright's route interception
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

