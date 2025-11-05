import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import AuthWrapper from './AuthWrapper'
import { useAuthStore } from '../store/auth.store'
import * as apiModule from '../lib/api'

vi.mock('../lib/api', () => ({
    api: {
        getAuthStatus: vi.fn(),
    },
}))

describe('AuthWrapper', () => {
    let queryClient: QueryClient

    beforeEach(() => {
        queryClient = new QueryClient({
            defaultOptions: {
                queries: {
                    retry: false,
                },
            },
        })
        useAuthStore.setState({ isSetupComplete: false })
        vi.clearAllMocks()
    })

    const renderAuthWrapper = (children: React.ReactNode) => {
        return render(
            <QueryClientProvider client={queryClient}>
                <BrowserRouter>
                    <Routes>
                        <Route path='/' element={<AuthWrapper>{children}</AuthWrapper>} />
                        <Route path='/settings' element={<div>Settings Page</div>} />
                    </Routes>
                </BrowserRouter>
            </QueryClientProvider>,
        )
    }

    it('should show loading state initially', () => {
        vi.mocked(apiModule.api.getAuthStatus).mockImplementation(
            () =>
                new Promise(() => {
                    // Never resolves
                }),
        )

        renderAuthWrapper(<div>Protected Content</div>)
        expect(screen.getByText('Loading V-Mail...')).toBeInTheDocument()
    })

    it('should render children when setup is complete', async () => {
        vi.mocked(apiModule.api.getAuthStatus).mockResolvedValue({
            isAuthenticated: true,
            isSetupComplete: true,
        })

        renderAuthWrapper(<div>Protected Content</div>)

        await waitFor(() => {
            expect(screen.getByText('Protected Content')).toBeInTheDocument()
        })
    })

    it('should redirect to settings when setup is not complete', async () => {
        vi.mocked(apiModule.api.getAuthStatus).mockResolvedValue({
            isAuthenticated: true,
            isSetupComplete: false,
        })

        renderAuthWrapper(<div>Protected Content</div>)

        await waitFor(() => {
            expect(screen.getByText('Settings Page')).toBeInTheDocument()
        })
    })

    it('should set setup to false on API error', async () => {
        vi.mocked(apiModule.api.getAuthStatus).mockRejectedValue(new Error('API Error'))

        renderAuthWrapper(<div>Protected Content</div>)

        await waitFor(() => {
            expect(useAuthStore.getState().isSetupComplete).toBe(false)
        })
    })
})
