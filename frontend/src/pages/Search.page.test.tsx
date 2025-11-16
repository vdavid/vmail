import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import * as React from 'react'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { describe, it, expect } from 'vitest'

import SearchPage from './Search.page'

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
                    <Route path='/search' element={children} />
                </Routes>
            </BrowserRouter>
        </QueryClientProvider>
    )
}

describe('SearchPage', () => {
    it('should render a loading state', async () => {
        window.history.pushState({}, '', '/search?q=test')
        render(<SearchPage />, { wrapper: createWrapper() })
        await waitFor(
            () => {
                const loadingText = screen.queryByText('Loading...')
                expect(loadingText).toBeInTheDocument()
            },
            { timeout: 1000 },
        )
    })

    it('should display search results', async () => {
        window.history.pushState({}, '', '/search?q=test')
        render(<SearchPage />, { wrapper: createWrapper() })

        await waitFor(() => {
            expect(screen.queryByText('Loading...')).not.toBeInTheDocument()
        })

        expect(screen.getByRole('heading', { name: 'Search results' })).toBeInTheDocument()
        expect(screen.getByText('"test"')).toBeInTheDocument()
        await waitFor(() => {
            expect(screen.getByText('Search Result 1')).toBeInTheDocument()
            expect(screen.getByText('Search Result 2')).toBeInTheDocument()
            expect(screen.getByText('Search Result 3')).toBeInTheDocument()
        })
    })

    it('should display empty state when no results', async () => {
        window.history.pushState({}, '', '/search?q=no-results')
        render(<SearchPage />, { wrapper: createWrapper() })

        await waitFor(() => {
            expect(screen.queryByText('Loading...')).not.toBeInTheDocument()
        })

        expect(screen.getByText(/No results found for "no-results"/)).toBeInTheDocument()
    })

    it('should display error state on API error', async () => {
        window.history.pushState({}, '', '/search?q=error-test')
        render(<SearchPage />, { wrapper: createWrapper() })

        await waitFor(() => {
            expect(screen.queryByText('Loading...')).not.toBeInTheDocument()
        })

        await waitFor(() => {
            expect(screen.getByText(/Error searching:/)).toBeInTheDocument()
        })
    })

    it('should handle empty query', async () => {
        window.history.pushState({}, '', '/search?q=')
        render(<SearchPage />, { wrapper: createWrapper() })

        await waitFor(() => {
            expect(screen.queryByText('Loading...')).not.toBeInTheDocument()
        })

        expect(screen.getByRole('heading', { name: 'Search' })).toBeInTheDocument()
        expect(screen.getByText('Enter a search query to find emails')).toBeInTheDocument()
    })

    it('should display query in header', async () => {
        window.history.pushState({}, '', '/search?q=my+search+query')
        render(<SearchPage />, { wrapper: createWrapper() })

        await waitFor(() => {
            expect(screen.queryByText('Loading...')).not.toBeInTheDocument()
        })

        expect(screen.getByRole('heading', { name: 'Search results' })).toBeInTheDocument()
        expect(screen.getByText('"my search query"')).toBeInTheDocument()
    })
})
