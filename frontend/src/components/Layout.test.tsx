import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import * as React from 'react'
import { BrowserRouter } from 'react-router-dom'
import { describe, it, expect } from 'vitest'

import Layout from './Layout'

describe('Layout', () => {
    const renderLayout = (children: React.ReactNode) => {
        const queryClient = new QueryClient({
            defaultOptions: {
                queries: { retry: false },
            },
        })
        return render(
            <QueryClientProvider client={queryClient}>
                <BrowserRouter>
                    <Layout>{children}</Layout>
                </BrowserRouter>
            </QueryClientProvider>,
        )
    }

    it('should render children', () => {
        renderLayout(<div>Test Content</div>)
        expect(screen.getByText('Test Content')).toBeInTheDocument()
    })

    it('should render Sidebar component', () => {
        renderLayout(<div>Test</div>)
        expect(screen.getByText('V-Mail')).toBeInTheDocument()
    })

    it('should render Header component', () => {
        renderLayout(<div>Test</div>)
        expect(screen.getByPlaceholderText('Search mail...')).toBeInTheDocument()
    })

    it('should have correct layout structure', () => {
        const { container } = renderLayout(<div>Test</div>)
        const mainLayout = container.firstChild
        expect(mainLayout).toHaveClass('flex', 'h-screen', 'overflow-hidden')
    })
})
