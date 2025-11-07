import { http, HttpResponse } from 'msw'

export const handlers = [
    http.get('/api/v1/folders', () => {
        return HttpResponse.json([{ name: 'INBOX' }, { name: 'Sent' }, { name: 'Drafts' }])
    }),

    http.get('/api/v1/threads', ({ request }) => {
        const url = new URL(request.url)
        const folder = url.searchParams.get('folder')

        if (folder === 'INBOX') {
            return HttpResponse.json([
                {
                    id: '1',
                    stable_thread_id: 'thread-1',
                    subject: 'Test Thread 1',
                    user_id: 'user-1',
                    messages: [
                        {
                            id: 'msg-1',
                            thread_id: '1',
                            user_id: 'user-1',
                            imap_uid: 1,
                            imap_folder_name: 'INBOX',
                            message_id_header: '<test1@example.com>',
                            from_address: 'sender1@example.com',
                            to_addresses: ['recipient@example.com'],
                            cc_addresses: [],
                            sent_at: '2025-01-01T00:00:00Z',
                            subject: 'Test Thread 1',
                            unsafe_body_html: '<p>Test message</p>',
                            body_text: 'Test message',
                            is_read: false,
                            is_starred: false,
                            attachments: [],
                        },
                    ],
                },
                {
                    id: '2',
                    stable_thread_id: 'thread-2',
                    subject: 'Test Thread 2',
                    user_id: 'user-1',
                    messages: [
                        {
                            id: 'msg-2',
                            thread_id: '2',
                            user_id: 'user-1',
                            imap_uid: 2,
                            imap_folder_name: 'INBOX',
                            message_id_header: '<test2@example.com>',
                            from_address: 'sender2@example.com',
                            to_addresses: ['recipient@example.com'],
                            cc_addresses: [],
                            sent_at: '2025-01-02T00:00:00Z',
                            subject: 'Test Thread 2',
                            unsafe_body_html: '<p>Test message 2</p>',
                            body_text: 'Test message 2',
                            is_read: true,
                            is_starred: true,
                            attachments: [],
                        },
                    ],
                },
            ])
        }

        return HttpResponse.json([])
    }),

    http.get('/api/v1/thread/:threadId', ({ params }) => {
        const threadId = params.threadId as string

        if (threadId === 'thread-1') {
            return HttpResponse.json({
                id: '1',
                stable_thread_id: 'thread-1',
                subject: 'Test Thread 1',
                user_id: 'user-1',
                messages: [
                    {
                        id: 'msg-1',
                        thread_id: '1',
                        user_id: 'user-1',
                        imap_uid: 1,
                        imap_folder_name: 'INBOX',
                        message_id_header: '<test1@example.com>',
                        from_address: 'sender1@example.com',
                        to_addresses: ['recipient@example.com'],
                        cc_addresses: [],
                        sent_at: '2025-01-01T00:00:00Z',
                        subject: 'Test Thread 1',
                        unsafe_body_html: '<p>Test message body</p>',
                        body_text: 'Test message body',
                        is_read: false,
                        is_starred: false,
                        attachments: [
                            {
                                id: 'att-1',
                                message_id: 'msg-1',
                                filename: 'test.pdf',
                                mime_type: 'application/pdf',
                                size_bytes: 1024,
                                is_inline: false,
                            },
                        ],
                    },
                ],
            })
        }

        return HttpResponse.json({ error: 'Thread not found' }, { status: 404 })
    }),
]
