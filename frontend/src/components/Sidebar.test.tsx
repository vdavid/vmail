import { describe, it, expect } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import Sidebar from './Sidebar'
import * as React from 'react'

const createWrapper = () => {
    const queryClient = new QueryClient({
        defaultOptions: {
            queries: { retry: false },
        },
    })
    return ({ children }: { children: React.ReactNode }) => (
        <QueryClientProvider client={queryClient}>
            <BrowserRouter>{children}</BrowserRouter>
        </QueryClientProvider>
    )
}

describe('Sidebar', () => {
    it('should render the V-Mail title', () => {
        render(<Sidebar />, { wrapper: createWrapper() })
        expect(screen.getByText('V-Mail')).toBeInTheDocument()
    })

    it('should render a loading state', () => {
        render(<Sidebar />, { wrapper: createWrapper() })
        expect(screen.getByText('Loading...')).toBeInTheDocument()
    })

    it('should call GET /api/v1/folders', async () => {
        render(<Sidebar />, { wrapper: createWrapper() })

        await waitFor(() => {
            expect(screen.queryByText('Loading...')).not.toBeInTheDocument()
        })
    })

    it('should render a list of links based on the mock API response', async () => {
        render(<Sidebar />, { wrapper: createWrapper() })

        await waitFor(() => {
            expect(screen.getByText('INBOX')).toBeInTheDocument()
            expect(screen.getByText('Sent')).toBeInTheDocument()
            expect(screen.getByText('Drafts')).toBeInTheDocument()
        })
    })

    it('should navigate to the correct folder when clicking a link', async () => {
        render(<Sidebar />, { wrapper: createWrapper() })

        await waitFor(() => {
            expect(screen.getByText('Sent')).toBeInTheDocument()
        })

        const sentLink = screen.getByText('Sent')
        expect(sentLink.closest('a')).toHaveAttribute('href', '/?folder=Sent')
    })

    it('should render Settings link', () => {
        render(<Sidebar />, { wrapper: createWrapper() })
        const settingsLink = screen.getByRole('link', { name: /Settings/i })
        expect(settingsLink).toHaveAttribute('href', '/settings')
    })
})
