import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import InboxPage from './Inbox.page'

describe('InboxPage', () => {
    const renderInboxPage = (initialPath: string = '/') => {
        window.history.pushState({}, '', initialPath)
        return render(
            <BrowserRouter>
                <Routes>
                    <Route path='/' element={<InboxPage />} />
                </Routes>
            </BrowserRouter>,
        )
    }

    it('should render Inbox title when folder is INBOX', () => {
        renderInboxPage('/?folder=INBOX')
        expect(screen.getByText('Inbox')).toBeInTheDocument()
    })

    it('should render default Inbox title when no folder param', () => {
        renderInboxPage('/')
        expect(screen.getByText('Inbox')).toBeInTheDocument()
    })

    it('should render folder name when folder param is provided', () => {
        renderInboxPage('/?folder=Starred')
        expect(screen.getByText('Starred')).toBeInTheDocument()
    })

    it('should render placeholder text', () => {
        renderInboxPage()
        expect(screen.getByText('Thread list will be displayed here.')).toBeInTheDocument()
    })
})
