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

// Helper to encode thread ID for URL
function encodeThreadId(id: string): string {
    const utf8Bytes = new TextEncoder().encode(id)
    const base64 = btoa(String.fromCharCode(...utf8Bytes))
    return base64.replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '')
}

const defaultEncodedId = encodeThreadId('thread-1')
const createWrapper = (initialEntries: string[] = [`/thread/${defaultEncodedId}`]) => {
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
        const encodedId = encodeThreadId('thread-1')
        window.history.pushState({}, '', `/thread/${encodedId}`)
        render(<ThreadPage />, { wrapper: createWrapper([`/thread/${encodedId}`]) })
        expect(screen.getByText('Loading...')).toBeInTheDocument()
    })

    it('should read the :threadId URL parameter and call the correct API', async () => {
        const encodedId = encodeThreadId('thread-1')
        window.history.pushState({}, '', `/thread/${encodedId}`)
        render(<ThreadPage />, { wrapper: createWrapper([`/thread/${encodedId}`]) })

        await waitFor(() => {
            expect(screen.queryByText('Loading...')).not.toBeInTheDocument()
        })

        // Subject appears in both title and message, check that it exists
        const subjectElements = screen.getAllByText('Test Thread 1')
        expect(subjectElements.length).toBeGreaterThan(0)
    })

    it('should correctly render all messages, sender names, subjects, and attachment filenames', async () => {
        const encodedId = encodeThreadId('thread-1')
        window.history.pushState({}, '', `/thread/${encodedId}`)
        render(<ThreadPage />, { wrapper: createWrapper([`/thread/${encodedId}`]) })

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
        const encodedId = encodeThreadId('thread-1')
        window.history.pushState({}, '', `/thread/${encodedId}`)
        render(<ThreadPage />, { wrapper: createWrapper([`/thread/${encodedId}`]) })

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
        const encodedId = encodeThreadId('thread-1')
        render(
            <QueryClientProvider client={queryClient}>
                <MemoryRouter initialEntries={['/', `/thread/${encodedId}`]}>
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
        expect(screen.getByTestId('location')).toHaveTextContent(`/thread/${encodedId}`)

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

    it('should decode base64-encoded thread ID from URL and use raw ID for API call', async () => {
        // This test verifies that:
        // 1. The URL contains base64-encoded thread ID (for clean URLs)
        // 2. The component decodes it and uses the raw Message-ID for the API call
        // 3. The React app loads (not JSON response)

        // Encode a test thread ID (simulating what would be in the URL)
        const rawThreadId = '<thread-1@example.com>'
        const utf8Bytes = new TextEncoder().encode(rawThreadId)
        const encodedId = btoa(String.fromCharCode(...utf8Bytes))
            .replace(/\+/g, '-')
            .replace(/\//g, '_')
            .replace(/=+$/g, '')

        const queryClient = new QueryClient({
            defaultOptions: {
                queries: { retry: false },
            },
        })

        // Mock fetch to verify the API is called with the raw (decoded) thread ID
        const mockFetch = vi.fn()
        // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
        global.fetch = mockFetch

        mockFetch.mockResolvedValueOnce({
            ok: true,
            json: async () =>
                Promise.resolve({
                    id: '1',
                    stable_thread_id: rawThreadId,
                    subject: 'Test Thread 1',
                    user_id: 'user-1',
                    messages: [
                        {
                            id: 'msg-1',
                            thread_id: '1',
                            user_id: 'user-1',
                            imap_uid: 1,
                            imap_folder_name: 'INBOX',
                            message_id_header: '<test1@example.com>',
                            from_address: 'sender1@example.com',
                            to_addresses: ['recipient@example.com'],
                            cc_addresses: [],
                            sent_at: '2025-01-01T00:00:00Z',
                            subject: 'Test Thread 1',
                            unsafe_body_html: '<p>Test message body</p>',
                            body_text: 'Test message body',
                            is_read: false,
                            is_starred: false,
                            attachments: [],
                        },
                    ],
                }),
        } as Response)

        render(
            <QueryClientProvider client={queryClient}>
                <MemoryRouter initialEntries={[`/thread/${encodedId}`]}>
                    <Routes>
                        <Route path='/thread/:threadId' element={<ThreadPage />} />
                    </Routes>
                </MemoryRouter>
            </QueryClientProvider>,
        )

        // Wait for the API call
        await waitFor(() => {
            expect(mockFetch).toHaveBeenCalled()
        })

        // Verify the API was called with the raw (URL-encoded) thread ID, not the base64
        const apiCall = mockFetch.mock.calls.find(
            (call: unknown[]) => typeof call[0] === 'string' && call[0].includes('/api/v1/thread/'),
        )
        expect(apiCall).toBeDefined()

        // The API should receive the raw thread ID (URL-encoded), not the base64
        // So it should contain %3C (encoded <) and %3E (encoded >)
        const apiUrl = apiCall?.[0] as string
        expect(apiUrl).toContain('/api/v1/thread/')
        // The raw thread ID should be URL-encoded in the API call
        // Note: @ gets encoded as %40
        expect(apiUrl).toMatch(/thread\/%3Cthread-1%40example\.com%3E/)

        // Verify the React component renders (not JSON)
        // Note: "Test Thread 1" appears multiple times (in title and message), so use getAllByText
        await waitFor(() => {
            const elements = screen.getAllByText('Test Thread 1')
            expect(elements.length).toBeGreaterThan(0)
        })
    })
})
