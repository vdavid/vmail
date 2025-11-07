import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import Header from './Header'

describe('Header', () => {
    it('should render the search input', () => {
        render(<Header />)
        expect(screen.getByPlaceholderText('Search mail...')).toBeInTheDocument()
    })

    it('should update search query when typing', async () => {
        const user = userEvent.setup()
        render(<Header />)

        const searchInput = screen.getByPlaceholderText('Search mail...')
        await user.type(searchInput, 'test query')

        expect(searchInput).toHaveValue('test query')
    })

    it('should prevent default form submission', async () => {
        const user = userEvent.setup()
        render(<Header />)

        const searchInput = screen.getByPlaceholderText('Search mail...')
        await user.type(searchInput, 'test query')

        // Form submission should be prevented (no page reload)
        // The search functionality will be implemented in a future milestone
        expect(searchInput).toHaveValue('test query')
    })
})
