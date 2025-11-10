/**
 * Test data fixtures for E2E tests.
 */

export interface TestUser {
    email: string
    imapServer: string
    imapUsername: string
    imapPassword: string
    smtpServer: string
    smtpUsername: string
    smtpPassword: string
}

export interface TestMessage {
    messageId: string
    subject: string
    from: string
    to: string
    body: string
    sentAt: Date
}

export const defaultTestUser: TestUser = {
    email: 'test@example.com',
    imapServer: 'localhost:1143', // Will be set by test server
    imapUsername: 'username',
    imapPassword: 'password',
    smtpServer: 'localhost:1025', // Will be set by test server
    smtpUsername: 'test-user',
    smtpPassword: 'test-pass',
}

export const sampleMessages: TestMessage[] = [
    {
        messageId: '<msg1@test>',
        subject: 'Welcome to V-Mail',
        from: 'sender@example.com',
        to: 'test@example.com',
        body: 'This is a test message.',
        sentAt: new Date(Date.now() - 2 * 60 * 60 * 1000), // 2 hours ago
    },
    {
        messageId: '<msg2@test>',
        subject: 'Meeting Tomorrow',
        from: 'colleague@example.com',
        to: 'test@example.com',
        body: 'Don\'t forget about the meeting tomorrow at 2 PM.',
        sentAt: new Date(Date.now() - 60 * 60 * 1000), // 1 hour ago
    },
    {
        messageId: '<msg3@test>',
        subject: 'Special Report Q3',
        from: 'reports@example.com',
        to: 'test@example.com',
        body: 'Here is the Q3 report you requested.',
        sentAt: new Date(), // Now
    },
]

