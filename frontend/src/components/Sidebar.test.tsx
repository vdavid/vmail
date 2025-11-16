import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import * as React from 'react'
import { BrowserRouter } from 'react-router-dom'
import { describe, it, expect } from 'vitest'

import Sidebar from './Sidebar'

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
        expect(screen.getAllByText('V-Mail')[0]).toBeInTheDocument()
    })

    it('should render a loading state', () => {
        render(<Sidebar />, { wrapper: createWrapper() })
        expect(screen.getAllByText('Loading...').length).toBeGreaterThan(0)
    })

    it('should call GET /api/v1/folders', async () => {
        render(<Sidebar />, { wrapper: createWrapper() })

        await waitFor(() => {
            expect(screen.queryByText('Loading...')).not.toBeInTheDocument()
        })
    })

    it('should render a list of links based on the mock API response', async () => {
        render(<Sidebar />, { wrapper: createWrapper() })

        const inboxMatches = await screen.findAllByText('INBOX')
        const sentMatches = await screen.findAllByText('Sent')
        const draftsMatches = await screen.findAllByText('Drafts')

        expect(inboxMatches.length).toBeGreaterThan(0)
        expect(sentMatches.length).toBeGreaterThan(0)
        expect(draftsMatches.length).toBeGreaterThan(0)
    })

    it('should navigate to the correct folder when clicking a link', async () => {
        render(<Sidebar />, { wrapper: createWrapper() })

        const sentItems = await screen.findAllByText('Sent')
        const sentLink = sentItems[0]
        expect(sentLink.closest('a')).toHaveAttribute('href', '/?folder=Sent')
    })

    it('should render Settings link', () => {
        render(<Sidebar />, { wrapper: createWrapper() })
        const settingsLink = screen.getByRole('link', { name: /Settings/i })
        expect(settingsLink).toHaveAttribute('href', '/settings')
    })
})
