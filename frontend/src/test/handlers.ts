import { http, HttpResponse } from 'msw'

export const handlers = [
    http.get('/api/v1/settings', () => {
        return HttpResponse.json({
            imap_server_hostname: 'imap.example.com',
            imap_username: 'user@example.com',
            imap_password: 'password',
            smtp_server_hostname: 'smtp.example.com',
            smtp_username: 'user@example.com',
            smtp_password: 'password',
            archive_folder_name: 'Archive',
            sent_folder_name: 'Sent',
            drafts_folder_name: 'Drafts',
            trash_folder_name: 'Trash',
            spam_folder_name: 'Spam',
            undo_send_delay_seconds: 20,
            pagination_threads_per_page: 100,
        })
    }),

    http.get('/api/v1/folders', () => {
        return HttpResponse.json([{ name: 'INBOX' }, { name: 'Sent' }, { name: 'Drafts' }])
    }),

    http.get('/api/v1/threads', ({ request }) => {
        const url = new URL(request.url)
        const folder = url.searchParams.get('folder')
        const page = parseInt(url.searchParams.get('page') || '1', 10)
        const limit = parseInt(url.searchParams.get('limit') || '100', 10)

        if (folder === 'INBOX') {
            const threads = [
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
            ]

            // Paginate the threads
            const startIndex = (page - 1) * limit
            const endIndex = startIndex + limit
            const paginatedThreads = threads.slice(startIndex, endIndex)

            return HttpResponse.json({
                threads: paginatedThreads,
                pagination: {
                    total_count: threads.length,
                    page: page,
                    per_page: limit,
                },
            })
        }

        return HttpResponse.json({
            threads: [],
            pagination: {
                total_count: 0,
                page: 1,
                per_page: limit,
            },
        })
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

    http.get('/api/v1/search', ({ request }) => {
        const url = new URL(request.url)
        const query = url.searchParams.get('q') || ''
        const page = parseInt(url.searchParams.get('page') || '1', 10)
        const limit = parseInt(url.searchParams.get('limit') || '100', 10)

        // Test invalid query (400 Bad Request)
        if (query === 'from:') {
            return HttpResponse.json(
                { error: 'invalid search query: empty from: value' },
                { status: 400 },
            )
        }

        // Test IMAP error (500 Internal Server Error)
        if (query === 'error-test') {
            return HttpResponse.json({ error: 'Internal server error' }, { status: 500 })
        }

        // Mock search results
        const allThreads = [
            {
                id: 'search-thread-1',
                stable_thread_id: 'search-stable-1',
                subject: 'Search Result 1',
                user_id: 'user-1',
                messages: [
                    {
                        id: 'search-msg-1',
                        thread_id: 'search-thread-1',
                        user_id: 'user-1',
                        imap_uid: 1,
                        imap_folder_name: 'INBOX',
                        message_id_header: '<search1@example.com>',
                        from_address: 'sender1@example.com',
                        to_addresses: ['recipient@example.com'],
                        cc_addresses: [],
                        sent_at: '2025-01-01T00:00:00Z',
                        subject: 'Search Result 1',
                        unsafe_body_html: '<p>Search message 1</p>',
                        body_text: 'Search message 1',
                        is_read: false,
                        is_starred: false,
                        attachments: [],
                    },
                ],
            },
            {
                id: 'search-thread-2',
                stable_thread_id: 'search-stable-2',
                subject: 'Search Result 2',
                user_id: 'user-1',
                messages: [
                    {
                        id: 'search-msg-2',
                        thread_id: 'search-thread-2',
                        user_id: 'user-1',
                        imap_uid: 2,
                        imap_folder_name: 'INBOX',
                        message_id_header: '<search2@example.com>',
                        from_address: 'sender2@example.com',
                        to_addresses: ['recipient@example.com'],
                        cc_addresses: [],
                        sent_at: '2025-01-02T00:00:00Z',
                        subject: 'Search Result 2',
                        unsafe_body_html: '<p>Search message 2</p>',
                        body_text: 'Search message 2',
                        is_read: false,
                        is_starred: false,
                        attachments: [],
                    },
                ],
            },
            {
                id: 'search-thread-3',
                stable_thread_id: 'search-stable-3',
                subject: 'Search Result 3',
                user_id: 'user-1',
                messages: [
                    {
                        id: 'search-msg-3',
                        thread_id: 'search-thread-3',
                        user_id: 'user-1',
                        imap_uid: 3,
                        imap_folder_name: 'INBOX',
                        message_id_header: '<search3@example.com>',
                        from_address: 'sender3@example.com',
                        to_addresses: ['recipient@example.com'],
                        cc_addresses: [],
                        sent_at: '2025-01-03T00:00:00Z',
                        subject: 'Search Result 3',
                        unsafe_body_html: '<p>Search message 3</p>',
                        body_text: 'Search message 3',
                        is_read: false,
                        is_starred: false,
                        attachments: [],
                    },
                ],
            },
        ]

        // Paginate
        const startIndex = (page - 1) * limit
        const endIndex = startIndex + limit
        const paginatedThreads = allThreads.slice(startIndex, endIndex)

        // Empty results for specific query
        if (query === 'no-results') {
            return HttpResponse.json({
                threads: [],
                pagination: {
                    total_count: 0,
                    page: page,
                    per_page: limit,
                },
            })
        }

        return HttpResponse.json({
            threads: paginatedThreads,
            pagination: {
                total_count: allThreads.length,
                page: page,
                per_page: limit,
            },
        })
    }),
]
