import { describe, it, expect, vi } from 'vitest'
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

    it('should call console.log on form submit', async () => {
        const consoleSpy = vi.spyOn(console, 'log')
        const user = userEvent.setup()
        render(<Header />)

        const searchInput = screen.getByPlaceholderText('Search mail...')
        await user.type(searchInput, 'test query')
        await user.type(searchInput, '{Enter}')

        expect(consoleSpy).toHaveBeenCalledWith('Search:', 'test query')
        consoleSpy.mockRestore()
    })
})
