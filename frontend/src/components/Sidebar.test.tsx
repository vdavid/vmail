import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import Sidebar from './Sidebar'

describe('Sidebar', () => {
    const renderSidebar = () => {
        return render(
            <BrowserRouter>
                <Sidebar />
            </BrowserRouter>,
        )
    }

    it('should render the V-Mail title', () => {
        renderSidebar()
        expect(screen.getByText('V-Mail')).toBeInTheDocument()
    })

    it('should render all navigation items', () => {
        renderSidebar()
        expect(screen.getByText('Inbox')).toBeInTheDocument()
        expect(screen.getByText('Starred')).toBeInTheDocument()
        expect(screen.getByText('Sent')).toBeInTheDocument()
        expect(screen.getByText('Drafts')).toBeInTheDocument()
        expect(screen.getByText('Spam')).toBeInTheDocument()
        expect(screen.getByText('Trash')).toBeInTheDocument()
        expect(screen.getByText('Settings')).toBeInTheDocument()
    })

    it('should render navigation links with correct hrefs', () => {
        renderSidebar()
        const inboxLink = screen.getByRole('link', { name: /Inbox/i })
        expect(inboxLink).toHaveAttribute('href', '/?folder=INBOX')

        const settingsLink = screen.getByRole('link', { name: /Settings/i })
        expect(settingsLink).toHaveAttribute('href', '/settings')
    })
})
