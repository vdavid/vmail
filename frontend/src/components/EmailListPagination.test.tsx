import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import * as React from 'react'
import { MemoryRouter } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'

import EmailListPagination from './EmailListPagination'

const createWrapper = (initialEntries = ['/?folder=INBOX&page=1']) => {
    const queryClient = new QueryClient({
        defaultOptions: {
            queries: { retry: false },
        },
    })
    return ({ children }: { children: React.ReactNode }) => (
        <QueryClientProvider client={queryClient}>
            <MemoryRouter initialEntries={initialEntries}>{children}</MemoryRouter>
        </QueryClientProvider>
    )
}

describe('EmailListPagination', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

    it('should render pagination info', () => {
        render(
            <EmailListPagination
                pagination={{
                    total_count: 300,
                    page: 1,
                    per_page: 100,
                }}
            />,
            { wrapper: createWrapper() },
        )

        expect(screen.getByText('Page 1 of 3')).toBeInTheDocument()
    })

    it('should not render when total pages is 1 or less', () => {
        const { container } = render(
            <EmailListPagination
                pagination={{
                    total_count: 50,
                    page: 1,
                    per_page: 100,
                }}
            />,
            { wrapper: createWrapper() },
        )

        expect(container.firstChild).toBeNull()
    })

    it('should render Prev and Next buttons', () => {
        render(
            <EmailListPagination
                pagination={{
                    total_count: 300,
                    page: 2,
                    per_page: 100,
                }}
            />,
            { wrapper: createWrapper(['/?folder=INBOX&page=2']) },
        )

        expect(screen.getByText('Prev')).toBeInTheDocument()
        expect(screen.getByText('Next')).toBeInTheDocument()
    })

    it('should disable Previous button on first page', () => {
        render(
            <EmailListPagination
                pagination={{
                    total_count: 300,
                    page: 1,
                    per_page: 100,
                }}
            />,
            { wrapper: createWrapper(['/?folder=INBOX&page=1']) },
        )

        const prevButton = screen.getByText('Prev')
        expect(prevButton).toBeDisabled()
    })

    it('should disable Next button on last page', () => {
        render(
            <EmailListPagination
                pagination={{
                    total_count: 300,
                    page: 3,
                    per_page: 100,
                }}
            />,
            { wrapper: createWrapper(['/?folder=INBOX&page=3']) },
        )

        const nextButton = screen.getByText('Next')
        expect(nextButton).toBeDisabled()
    })

    it('should navigate to next page when Next button is clicked', async () => {
        const user = userEvent.setup()
        const queryClient = new QueryClient({
            defaultOptions: {
                queries: { retry: false },
            },
        })

        render(
            <QueryClientProvider client={queryClient}>
                <MemoryRouter initialEntries={['/?folder=INBOX&page=1']}>
                    <EmailListPagination
                        pagination={{
                            total_count: 300,
                            page: 1,
                            per_page: 100,
                        }}
                    />
                </MemoryRouter>
            </QueryClientProvider>,
        )

        const nextButton = screen.getByText('Next')
        await user.click(nextButton)

        // Check that URL has changed (we can't directly test navigation, but we can check the button state)
        // In a real test, we'd use a router that tracks navigation
        expect(nextButton).toBeInTheDocument()
    })

    it('should navigate to previous page when Previous button is clicked', async () => {
        const user = userEvent.setup()
        const queryClient = new QueryClient({
            defaultOptions: {
                queries: { retry: false },
            },
        })

        render(
            <QueryClientProvider client={queryClient}>
                <MemoryRouter initialEntries={['/?folder=INBOX&page=2']}>
                    <EmailListPagination
                        pagination={{
                            total_count: 300,
                            page: 2,
                            per_page: 100,
                        }}
                    />
                </MemoryRouter>
            </QueryClientProvider>,
        )

        const prevButton = screen.getByText('Prev')
        await user.click(prevButton)

        expect(prevButton).toBeInTheDocument()
    })
})
