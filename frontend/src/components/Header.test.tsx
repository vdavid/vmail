import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BrowserRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import Header from './Header'

// Mock useNavigate
const mockNavigate = vi.fn()
vi.mock('react-router-dom', async () => {
    const actual = await vi.importActual('react-router-dom')
    return {
        ...actual,
        useNavigate: () => mockNavigate,
    }
})

describe('Header', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

    it('should render the search input', () => {
        render(
            <BrowserRouter>
                <Header />
            </BrowserRouter>,
        )
        expect(screen.getByPlaceholderText('Search mail...')).toBeInTheDocument()
    })

    it('should update search query when typing', async () => {
        const user = userEvent.setup()
        render(
            <BrowserRouter>
                <Header />
            </BrowserRouter>,
        )

        const searchInput = screen.getByPlaceholderText('Search mail...')
        await user.type(searchInput, 'test query')

        expect(searchInput).toHaveValue('test query')
    })

    it('should navigate to search page on form submission', async () => {
        const user = userEvent.setup()
        render(
            <BrowserRouter>
                <Header />
            </BrowserRouter>,
        )

        const searchInput = screen.getByPlaceholderText('Search mail...')
        await user.type(searchInput, 'test query')
        await user.keyboard('{Enter}')

        expect(mockNavigate).toHaveBeenCalledWith('/search?q=test%20query')
    })

    it('should show validation error for invalid query', async () => {
        const user = userEvent.setup()
        render(
            <BrowserRouter>
                <Header />
            </BrowserRouter>,
        )

        const searchInput = screen.getByPlaceholderText('Search mail...')
        await user.type(searchInput, 'from:')
        await user.keyboard('{Enter}')

        expect(mockNavigate).not.toHaveBeenCalled()
        expect(screen.getByText(/empty/i, { selector: '[role="alert"]' })).toBeInTheDocument()
    })

    it('should clear validation error when user types', async () => {
        const user = userEvent.setup()
        render(
            <BrowserRouter>
                <Header />
            </BrowserRouter>,
        )

        const searchInput = screen.getByPlaceholderText('Search mail...')
        await user.type(searchInput, 'from:')
        await user.keyboard('{Enter}')

        expect(screen.getByText(/empty/i, { selector: '[role="alert"]' })).toBeInTheDocument()

        await user.type(searchInput, 'george')

        expect(screen.queryByText(/empty/i, { selector: '[role="alert"]' })).not.toBeInTheDocument()
    })
})
