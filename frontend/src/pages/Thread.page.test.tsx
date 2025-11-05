import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import userEvent from '@testing-library/user-event'
import ThreadPage from './Thread.page'

describe('ThreadPage', () => {
    const renderThreadPage = (threadId: string = '123') => {
        window.history.pushState({}, '', `/thread/${threadId}`)
        return render(
            <BrowserRouter>
                <Routes>
                    <Route path='/thread/:threadId' element={<ThreadPage />} />
                    <Route path='/' element={<div>Inbox Page</div>} />
                </Routes>
            </BrowserRouter>,
        )
    }

    it('should render thread ID from URL params', () => {
        renderThreadPage('test-thread-123')
        expect(screen.getByText('Thread: test-thread-123')).toBeInTheDocument()
    })

    it('should render back button', () => {
        renderThreadPage()
        expect(screen.getByRole('button', { name: /Back to Inbox/i })).toBeInTheDocument()
    })

    it('should navigate back to inbox when back button is clicked', async () => {
        const user = userEvent.setup()
        renderThreadPage()

        const backButton = screen.getByRole('button', {
            name: /Back to Inbox/i,
        })
        await user.click(backButton)

        expect(screen.getByText('Inbox Page')).toBeInTheDocument()
    })

    it('should render placeholder text', () => {
        renderThreadPage()
        expect(screen.getByText('Thread messages will be displayed here.')).toBeInTheDocument()
    })
})
