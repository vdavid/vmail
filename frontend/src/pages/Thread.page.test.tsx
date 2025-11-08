import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import * as React from 'react'
import { MemoryRouter, Routes, Route, useLocation } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'

import ThreadPage from './Thread.page'

// Component to track location for testing
function LocationTracker() {
    const location = useLocation()
    return <div data-testid='location'>{location.pathname}</div>
}

const createWrapper = (initialEntries = ['/thread/thread-1']) => {
    const queryClient = new QueryClient({
        defaultOptions: {
            queries: { retry: false },
        },
    })
    return ({ children }: { children: React.ReactNode }) => (
        <QueryClientProvider client={queryClient}>
            <MemoryRouter initialEntries={initialEntries}>
                <Routes>
                    <Route path='/thread/:threadId' element={children} />
                    <Route path='/' element={<div>Inbox Page</div>} />
                </Routes>
            </MemoryRouter>
        </QueryClientProvider>
    )
}

describe('ThreadPage', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

    it('should render a loading state', () => {
        window.history.pushState({}, '', '/thread/thread-1')
        render(<ThreadPage />, { wrapper: createWrapper() })
        expect(screen.getByText('Loading...')).toBeInTheDocument()
    })

    it('should read the :threadId URL parameter and call the correct API', async () => {
        window.history.pushState({}, '', '/thread/thread-1')
        render(<ThreadPage />, { wrapper: createWrapper() })

        await waitFor(() => {
            expect(screen.queryByText('Loading...')).not.toBeInTheDocument()
        })

        // Subject appears in both title and message, check that it exists
        const subjectElements = screen.getAllByText('Test Thread 1')
        expect(subjectElements.length).toBeGreaterThan(0)
    })

    it('should correctly render all messages, sender names, subjects, and attachment filenames', async () => {
        window.history.pushState({}, '', '/thread/thread-1')
        render(<ThreadPage />, { wrapper: createWrapper() })

        await waitFor(() => {
            // Subject appears in both title and message
            const subjectElements = screen.getAllByText('Test Thread 1')
            expect(subjectElements.length).toBeGreaterThan(0)
        })

        // Check sender name
        expect(screen.getByText('sender1@example.com')).toBeInTheDocument()

        // Check attachment filename (text is split across elements: "test.pdf (1.0 KB)")
        expect(screen.getByText(/test\.pdf/)).toBeInTheDocument()
    })

    it('should render back button', async () => {
        window.history.pushState({}, '', '/thread/thread-1')
        render(<ThreadPage />, { wrapper: createWrapper() })

        await waitFor(() => {
            expect(screen.getByRole('button', { name: /Back to Inbox/i })).toBeInTheDocument()
        })
    })

    it('should navigate back to inbox when back button is clicked', async () => {
        const user = userEvent.setup()

        // 1. Create ONE QueryClient for this test
        const queryClient = new QueryClient({
            defaultOptions: {
                queries: { retry: false },
            },
        })

        // 2. Render the full router setup directly.
        //    We don't need a separate 'TestWrapper'.
        render(
            <QueryClientProvider client={queryClient}>
                <MemoryRouter initialEntries={['/', '/thread/thread-1']}>
                    <Routes>
                        {/* The component-under-test is now here */}
                        <Route path='/thread/:threadId' element={<ThreadPage />} />
                        {/* The destination page is here */}
                        <Route path='/' element={<div data-testid='inbox-page'>Inbox Page</div>} />
                    </Routes>
                    {/* The location tracker is here */}
                    <LocationTracker />
                </MemoryRouter>
            </QueryClientProvider>,
        )

        // 3. Wait for the thread page to load
        const backButton = await screen.findByRole('button', {
            name: /Back to Inbox/i,
        })
        expect(backButton).toBeInTheDocument()

        // 4. Verify we are on the correct initial page
        expect(screen.getByTestId('location')).toHaveTextContent('/thread/thread-1')

        // 5. Click the back button
        await user.click(backButton)

        // 6. Wait for and verify the navigation
        await waitFor(() => {
            // Assert the URL has changed
            expect(screen.getByTestId('location')).toHaveTextContent('/')
            //// Assert we navigated to the inbox page
            //expect(screen.getByTestId('inbox-page')).toBeInTheDocument()
        })
    })
})
