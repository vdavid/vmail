import { render } from '@testing-library/react'
import DOMPurify from 'dompurify'
import { describe, it, expect, vi, beforeEach } from 'vitest'

import type { Message as MessageType } from '../lib/api'

import Message from './Message'

// Mock DOMPurify
vi.mock('dompurify', () => ({
    default: {
        sanitize: vi.fn((html: string) => `sanitized:${html}`),
    },
}))

describe('Message', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

    it('always calls DOMPurify.sanitize with unsafe_body_html', () => {
        const message: MessageType = {
            id: '1',
            thread_id: 'thread-1',
            user_id: 'user-1',
            imap_uid: 1,
            imap_folder_name: 'INBOX',
            message_id_header: '<test@example.com>',
            from_address: 'sender@example.com',
            to_addresses: ['recipient@example.com'],
            cc_addresses: [],
            sent_at: '2025-01-01T00:00:00Z',
            subject: 'Test Subject',
            unsafe_body_html: '<p>Test HTML</p><script>alert("xss")</script>',
            body_text: 'Test text',
            is_read: false,
            is_starred: false,
            attachments: [],
        }

        render(<Message message={message} />)

        // eslint-disable-next-line @typescript-eslint/unbound-method
        expect(DOMPurify.sanitize).toHaveBeenCalledWith(
            '<p>Test HTML</p><script>alert("xss")</script>',
        )
    })

    it('renders sanitized HTML via dangerouslySetInnerHTML', () => {
        const message: MessageType = {
            id: '1',
            thread_id: 'thread-1',
            user_id: 'user-1',
            imap_uid: 1,
            imap_folder_name: 'INBOX',
            message_id_header: '<test@example.com>',
            from_address: 'sender@example.com',
            to_addresses: ['recipient@example.com'],
            cc_addresses: [],
            sent_at: '2025-01-01T00:00:00Z',
            subject: 'Test Subject',
            unsafe_body_html: '<p>Test HTML</p>',
            body_text: '',
            is_read: false,
            is_starred: false,
            attachments: [],
        }

        const { container } = render(<Message message={message} />)

        // Check that the sanitized content is rendered
        // eslint-disable-next-line @typescript-eslint/unbound-method
        expect(DOMPurify.sanitize).toHaveBeenCalled()
        const messageContent = container.querySelector('.prose')
        expect(messageContent).toBeTruthy()
        expect(messageContent?.innerHTML).toBe('sanitized:<p>Test HTML</p>')
    })

    it('renders body_text when unsafe_body_html is empty', () => {
        const message: MessageType = {
            id: '1',
            thread_id: 'thread-1',
            user_id: 'user-1',
            imap_uid: 1,
            imap_folder_name: 'INBOX',
            message_id_header: '<test@example.com>',
            from_address: 'sender@example.com',
            to_addresses: ['recipient@example.com'],
            cc_addresses: [],
            sent_at: '2025-01-01T00:00:00Z',
            subject: 'Test Subject',
            unsafe_body_html: '',
            body_text: 'Plain text body',
            is_read: false,
            is_starred: false,
            attachments: [],
        }

        const { getByText } = render(<Message message={message} />)

        expect(getByText('Plain text body')).toBeTruthy()
    })

    it('does not call DOMPurify when unsafe_body_html is empty', () => {
        const message: MessageType = {
            id: '1',
            thread_id: 'thread-1',
            user_id: 'user-1',
            imap_uid: 1,
            imap_folder_name: 'INBOX',
            message_id_header: '<test@example.com>',
            from_address: 'sender@example.com',
            to_addresses: ['recipient@example.com'],
            cc_addresses: [],
            sent_at: '2025-01-01T00:00:00Z',
            subject: 'Test Subject',
            unsafe_body_html: '',
            body_text: 'Plain text body',
            is_read: false,
            is_starred: false,
            attachments: [],
        }

        render(<Message message={message} />)

        // DOMPurify should not be called when unsafe_body_html is empty
        // eslint-disable-next-line @typescript-eslint/unbound-method
        expect(DOMPurify.sanitize).not.toHaveBeenCalled()
    })
})
