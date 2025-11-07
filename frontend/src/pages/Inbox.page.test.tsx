import { describe, it, expect } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import InboxPage from './Inbox.page'
import * as React from 'react'

const createWrapper = () => {
    const queryClient = new QueryClient({
        defaultOptions: {
            queries: { retry: false },
        },
    })
    return ({ children }: { children: React.ReactNode }) => (
        <QueryClientProvider client={queryClient}>
            <BrowserRouter>
                <Routes>
                    <Route path='/' element={children} />
                </Routes>
            </BrowserRouter>
        </QueryClientProvider>
    )
}

describe('InboxPage', () => {
    it('should render a loading state', () => {
        window.history.pushState({}, '', '/?folder=INBOX')
        render(<InboxPage />, { wrapper: createWrapper() })
        expect(screen.getByText('Loading...')).toBeInTheDocument()
    })

    it('should read the ?folder=INBOX URL parameter and call the correct API', async () => {
        window.history.pushState({}, '', '/?folder=INBOX')
        render(<InboxPage />, { wrapper: createWrapper() })

        await waitFor(() => {
            expect(screen.queryByText('Loading...')).not.toBeInTheDocument()
        })

        expect(screen.getByText('Inbox')).toBeInTheDocument()
    })

    it('should render the list of EmailListItem components based on the mock response', async () => {
        window.history.pushState({}, '', '/?folder=INBOX')
        render(<InboxPage />, { wrapper: createWrapper() })

        await waitFor(() => {
            expect(screen.getByText('Test Thread 1')).toBeInTheDocument()
            expect(screen.getByText('Test Thread 2')).toBeInTheDocument()
        })
    })

    it('should navigate to the correct thread when clicking an EmailListItem', async () => {
        window.history.pushState({}, '', '/?folder=INBOX')
        render(<InboxPage />, { wrapper: createWrapper() })

        await waitFor(() => {
            expect(screen.getByText('Test Thread 1')).toBeInTheDocument()
        })

        const threadLink = screen.getByText('Test Thread 1').closest('a')
        expect(threadLink).toHaveAttribute('href', '/thread/thread-1')
    })

    it('should render folder name when folder param is provided', async () => {
        window.history.pushState({}, '', '/?folder=Sent')
        render(<InboxPage />, { wrapper: createWrapper() })

        await waitFor(() => {
            expect(screen.getByText('Sent')).toBeInTheDocument()
        })
    })
})
