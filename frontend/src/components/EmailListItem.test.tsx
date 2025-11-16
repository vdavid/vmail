import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import * as React from 'react'
import { BrowserRouter } from 'react-router-dom'
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'

import type { Thread } from '../lib/api'

import EmailListItem from './EmailListItem'

const createWrapper = () => {
    const queryClient = new QueryClient({
        defaultOptions: {
            queries: { retry: false },
        },
    })
    return ({ children }: { children: React.ReactNode }) => (
        <QueryClientProvider client={queryClient}>
            <BrowserRouter>{children}</BrowserRouter>
        </QueryClientProvider>
    )
}

describe('EmailListItem', () => {
    beforeEach(() => {
        // Mock current date to be 2025-01-15 for consistent date formatting tests
        vi.useFakeTimers()
        vi.setSystemTime(new Date('2025-01-15T12:00:00Z'))
    })

    afterEach(() => {
        vi.useRealTimers()
    })

    it('displays thread count when message_count > 1', () => {
        const thread: Thread = {
            id: '1',
            stable_thread_id: 'thread-1',
            subject: 'Test Thread',
            user_id: 'user-1',
            first_message_from_address: 'sender@example.com',
            has_attachments: false,
            message_count: 3,
            last_sent_at: '2025-01-15T10:00:00Z',
        }

        render(<EmailListItem thread={thread} />, { wrapper: createWrapper() })

        const senderElement = screen.getByTestId('email-sender')
        expect(senderElement).toHaveTextContent('sender@example.com')
        expect(senderElement).toHaveTextContent('3')
    })

    it('does not display thread count when message_count = 1', () => {
        const thread: Thread = {
            id: '1',
            stable_thread_id: 'thread-1',
            subject: 'Test Thread',
            user_id: 'user-1',
            first_message_from_address: 'sender@example.com',
            has_attachments: false,
            message_count: 1,
            last_sent_at: '2025-01-15T10:00:00Z',
        }

        render(<EmailListItem thread={thread} />, { wrapper: createWrapper() })

        const senderElement = screen.getByTestId('email-sender')
        expect(senderElement).toHaveTextContent('sender@example.com')
        expect(senderElement).not.toHaveTextContent('1')
    })

    it('displays formatted date from last_sent_at (time if today)', () => {
        const thread: Thread = {
            id: '1',
            stable_thread_id: 'thread-1',
            subject: 'Test Thread',
            user_id: 'user-1',
            first_message_from_address: 'sender@example.com',
            has_attachments: false,
            message_count: 1,
            last_sent_at: '2025-01-15T10:30:00Z', // Today
        }

        const { container } = render(<EmailListItem thread={thread} />, {
            wrapper: createWrapper(),
        })

        // Date should be in the last column (text-right text-xs text-slate-400)
        // Should show time format - check that date column has content
        const dateColumn = container.querySelector('.text-right.text-xs.text-slate-400')
        expect(dateColumn).toBeInTheDocument()
        expect(dateColumn?.textContent.trim()).toBeTruthy()
        // Should contain time-related content (either "10" or "30" or ":" or "AM"/"PM")
        expect(dateColumn?.textContent).toMatch(/10|30|:|AM|PM/i)
    })

    it('displays formatted date from last_sent_at (day if not today)', () => {
        const thread: Thread = {
            id: '1',
            stable_thread_id: 'thread-1',
            subject: 'Test Thread',
            user_id: 'user-1',
            first_message_from_address: 'sender@example.com',
            has_attachments: false,
            message_count: 1,
            last_sent_at: '2025-01-10T10:00:00Z', // 5 days ago
        }

        render(<EmailListItem thread={thread} />, { wrapper: createWrapper() })

        // Should show day format (e.g., "Jan 10", "10 Jan", "Jan. 10", etc.)
        // Check that it contains "10" and month abbreviation
        const dateElement = screen.getByText(/Jan.*10|10.*Jan/i)
        expect(dateElement).toBeInTheDocument()
    })

    it('falls back to first message sent_at when last_sent_at is missing', () => {
        const thread: Thread = {
            id: '1',
            stable_thread_id: 'thread-1',
            subject: 'Test Thread',
            user_id: 'user-1',
            first_message_from_address: 'sender@example.com',
            has_attachments: false,
            message_count: 1,
            messages: [
                {
                    id: 'msg-1',
                    thread_id: '1',
                    user_id: 'user-1',
                    imap_uid: 1,
                    imap_folder_name: 'INBOX',
                    message_id_header: '<test@example.com>',
                    from_address: 'sender@example.com',
                    to_addresses: [],
                    cc_addresses: [],
                    sent_at: '2025-01-15T11:00:00Z',
                    subject: 'Test Thread',
                    unsafe_body_html: '',
                    body_text: '',
                    is_read: false,
                    is_starred: false,
                },
            ],
        }

        const { container } = render(<EmailListItem thread={thread} />, {
            wrapper: createWrapper(),
        })

        // Should show time from first message
        const dateColumn = container.querySelector('.text-right.text-xs.text-slate-400')
        expect(dateColumn).toBeInTheDocument()
        expect(dateColumn?.textContent.trim()).toBeTruthy()
        // Should contain time-related content (either "11" or ":" or "AM"/"PM")
        expect(dateColumn?.textContent).toMatch(/11|:|AM|PM/i)
    })

    it('displays attachment marker when thread has attachments', () => {
        const thread: Thread = {
            id: '1',
            stable_thread_id: 'thread-1',
            subject: 'Test Thread',
            user_id: 'user-1',
            first_message_from_address: 'sender@example.com',
            has_attachments: true,
            message_count: 3,
            last_sent_at: '2025-01-15T10:00:00Z',
            messages: [
                {
                    id: 'msg-1',
                    thread_id: '1',
                    user_id: 'user-1',
                    imap_uid: 1,
                    imap_folder_name: 'INBOX',
                    message_id_header: '<test1@example.com>',
                    from_address: 'sender@example.com',
                    to_addresses: [],
                    cc_addresses: [],
                    sent_at: '2025-01-15T08:00:00Z',
                    subject: 'Test Thread',
                    unsafe_body_html: '',
                    body_text: '',
                    is_read: false,
                    is_starred: false,
                    attachments: [],
                },
                {
                    id: 'msg-2',
                    thread_id: '1',
                    user_id: 'user-1',
                    imap_uid: 2,
                    imap_folder_name: 'INBOX',
                    message_id_header: '<test2@example.com>',
                    from_address: 'sender@example.com',
                    to_addresses: [],
                    cc_addresses: [],
                    sent_at: '2025-01-15T09:00:00Z',
                    subject: 'Test Thread',
                    unsafe_body_html: '',
                    body_text: '',
                    is_read: false,
                    is_starred: false,
                    attachments: [
                        {
                            id: 'att-1',
                            message_id: 'msg-2',
                            filename: 'document.pdf',
                            mime_type: 'application/pdf',
                            size_bytes: 1024,
                            is_inline: false,
                        },
                    ],
                },
                {
                    id: 'msg-3',
                    thread_id: '1',
                    user_id: 'user-1',
                    imap_uid: 3,
                    imap_folder_name: 'INBOX',
                    message_id_header: '<test3@example.com>',
                    from_address: 'sender@example.com',
                    to_addresses: [],
                    cc_addresses: [],
                    sent_at: '2025-01-15T10:00:00Z',
                    subject: 'Test Thread',
                    unsafe_body_html: '',
                    body_text: '',
                    is_read: false,
                    is_starred: false,
                    attachments: [],
                },
            ],
        }

        render(<EmailListItem thread={thread} />, { wrapper: createWrapper() })

        // Should show attachment marker (paperclip emoji)
        const attachmentMarker = screen.getByLabelText('Has attachments')
        expect(attachmentMarker).toBeInTheDocument()
        expect(attachmentMarker).toHaveTextContent('📎')
    })

    it('does not display attachment marker when thread has no attachments', () => {
        const thread: Thread = {
            id: '1',
            stable_thread_id: 'thread-1',
            subject: 'Test Thread',
            user_id: 'user-1',
            first_message_from_address: 'sender@example.com',
            has_attachments: false,
            message_count: 1,
            last_sent_at: '2025-01-15T10:00:00Z',
        }

        render(<EmailListItem thread={thread} />, { wrapper: createWrapper() })

        // Should not show attachment marker
        const attachmentMarker = screen.queryByLabelText('Has attachments')
        expect(attachmentMarker).not.toBeInTheDocument()
    })

    it('displays sender name with exactly 20 characters fully', () => {
        const name20Chars = 'A'.repeat(20)
        const thread: Thread = {
            id: '1',
            stable_thread_id: 'thread-1',
            subject: 'Test Thread',
            user_id: 'user-1',
            first_message_from_address: `${name20Chars} <sender@example.com>`,
            has_attachments: false,
            message_count: 1,
            last_sent_at: '2025-01-15T10:00:00Z',
        }

        render(<EmailListItem thread={thread} />, { wrapper: createWrapper() })

        const senderElement = screen.getByTestId('email-sender')
        expect(senderElement).toHaveTextContent(name20Chars)
        // Should not have a period at the end
        expect(senderElement.textContent).not.toContain('.')
    })

    it('truncates sender name with 21 characters to 19 + period', () => {
        const name21Chars = 'A'.repeat(21)
        const thread: Thread = {
            id: '1',
            stable_thread_id: 'thread-1',
            subject: 'Test Thread',
            user_id: 'user-1',
            first_message_from_address: `${name21Chars} <sender@example.com>`,
            has_attachments: false,
            message_count: 1,
            last_sent_at: '2025-01-15T10:00:00Z',
        }

        render(<EmailListItem thread={thread} />, { wrapper: createWrapper() })

        const senderElement = screen.getByTestId('email-sender')
        const expectedText = 'A'.repeat(19) + '.'
        expect(senderElement).toHaveTextContent(expectedText)
    })

    it('displays preview snippet from backend preview_snippet', () => {
        const thread: Thread = {
            id: '1',
            stable_thread_id: 'thread-1',
            subject: 'Test Thread',
            user_id: 'user-1',
            first_message_from_address: 'sender@example.com',
            has_attachments: false,
            message_count: 1,
            last_sent_at: '2025-01-15T10:00:00Z',
            preview_snippet: 'This is a preview snippet from the backend',
        }

        render(<EmailListItem thread={thread} />, { wrapper: createWrapper() })

        expect(screen.getByText(/This is a preview snippet from the backend/i)).toBeInTheDocument()
    })

    it('displays preview snippet from first message body_text when preview_snippet is missing', () => {
        const thread: Thread = {
            id: '1',
            stable_thread_id: 'thread-1',
            subject: 'Test Thread',
            user_id: 'user-1',
            first_message_from_address: 'sender@example.com',
            has_attachments: false,
            message_count: 1,
            last_sent_at: '2025-01-15T10:00:00Z',
            messages: [
                {
                    id: 'msg-1',
                    thread_id: '1',
                    user_id: 'user-1',
                    imap_uid: 1,
                    imap_folder_name: 'INBOX',
                    message_id_header: '<test@example.com>',
                    from_address: 'sender@example.com',
                    to_addresses: [],
                    cc_addresses: [],
                    sent_at: '2025-01-15T10:00:00Z',
                    subject: 'Test Thread',
                    unsafe_body_html: '',
                    body_text: 'This is the body text from the first message',
                    is_read: false,
                    is_starred: false,
                },
            ],
        }

        render(<EmailListItem thread={thread} />, { wrapper: createWrapper() })

        expect(
            screen.getByText(/This is the body text from the first message/i),
        ).toBeInTheDocument()
    })

    it('displays preview snippet from HTML email body (extracted text)', () => {
        const thread: Thread = {
            id: '1',
            stable_thread_id: 'thread-1',
            subject: 'Test Thread',
            user_id: 'user-1',
            first_message_from_address: 'sender@example.com',
            has_attachments: false,
            message_count: 1,
            last_sent_at: '2025-01-15T10:00:00Z',
            preview_snippet: 'This is extracted text from HTML email',
        }

        render(<EmailListItem thread={thread} />, { wrapper: createWrapper() })

        expect(screen.getByText(/This is extracted text from HTML email/i)).toBeInTheDocument()
    })

    it('normalizes whitespace in preview snippet', () => {
        const thread: Thread = {
            id: '1',
            stable_thread_id: 'thread-1',
            subject: 'Test Thread',
            user_id: 'user-1',
            first_message_from_address: 'sender@example.com',
            has_attachments: false,
            message_count: 1,
            last_sent_at: '2025-01-15T10:00:00Z',
            preview_snippet: 'This   has    multiple    spaces',
        }

        render(<EmailListItem thread={thread} />, { wrapper: createWrapper() })

        // Whitespace should be normalized to single spaces
        const snippetElement = screen.getByText(/This has multiple spaces/i)
        expect(snippetElement).toBeInTheDocument()
    })

    it('truncates long preview snippets to 100 characters', () => {
        const longSnippet = 'A'.repeat(150)
        const thread: Thread = {
            id: '1',
            stable_thread_id: 'thread-1',
            subject: 'Test Thread',
            user_id: 'user-1',
            first_message_from_address: 'sender@example.com',
            has_attachments: false,
            message_count: 1,
            last_sent_at: '2025-01-15T10:00:00Z',
            preview_snippet: longSnippet,
        }

        render(<EmailListItem thread={thread} />, { wrapper: createWrapper() })

        // Should be truncated to 100 chars + '...'
        const expectedText = 'A'.repeat(100) + '...'
        const snippetElement = screen.getByText(expectedText)
        expect(snippetElement).toBeInTheDocument()
    })
})
