import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import SettingsPage from './Settings.page'
import * as apiModule from '../lib/api'
import type { UserSettings } from '../lib/api'

vi.mock('../lib/api', () => ({
    api: {
        getSettings: vi.fn(),
        saveSettings: vi.fn(),
    },
}))

const mockSettings: UserSettings = {
    imap_server_hostname: 'imap.example.com:993',
    imap_username: 'user@example.com',
    imap_password: 'password123',
    smtp_server_hostname: 'smtp.example.com:587',
    smtp_username: 'user@example.com',
    smtp_password: 'password123',
    archive_folder_name: 'Archive',
    sent_folder_name: 'Sent',
    drafts_folder_name: 'Drafts',
    trash_folder_name: 'Trash',
    spam_folder_name: 'Spam',
    undo_send_delay_seconds: 20,
    pagination_threads_per_page: 100,
}

describe('SettingsPage', () => {
    let queryClient: QueryClient

    beforeEach(() => {
        queryClient = new QueryClient({
            defaultOptions: {
                queries: {
                    retry: false,
                },
            },
        })
        vi.clearAllMocks()
    })

    const renderSettingsPage = () => {
        return render(
            <QueryClientProvider client={queryClient}>
                <SettingsPage />
            </QueryClientProvider>,
        )
    }

    it('should show loading state initially', () => {
        vi.mocked(apiModule.api.getSettings).mockImplementation(
            () =>
                new Promise(() => {
                    // Never resolves
                }),
        )

        renderSettingsPage()
        expect(screen.getByText('Loading settings...')).toBeInTheDocument()
    })

    it('should render settings form', async () => {
        vi.mocked(apiModule.api.getSettings).mockRejectedValue(new Error('Not found'))

        renderSettingsPage()

        await waitFor(() => {
            expect(screen.getByText('Settings')).toBeInTheDocument()
        })

        expect(screen.getByLabelText('IMAP Server')).toBeInTheDocument()
        expect(screen.getByLabelText('IMAP Username')).toBeInTheDocument()
        expect(screen.getByLabelText('SMTP Server')).toBeInTheDocument()
    })

    it('should load existing settings', async () => {
        vi.mocked(apiModule.api.getSettings).mockResolvedValue(mockSettings)

        renderSettingsPage()

        await waitFor(() => {
            expect(screen.getByDisplayValue('imap.example.com:993')).toBeInTheDocument()
        })

        const userFields = screen.getAllByDisplayValue('user@example.com')
        expect(userFields).toHaveLength(2)
        expect(screen.getByDisplayValue('Archive')).toBeInTheDocument()
    })

    it('should handle form input changes', async () => {
        vi.mocked(apiModule.api.getSettings).mockRejectedValue(new Error('Not found'))

        const user = userEvent.setup()
        renderSettingsPage()

        await waitFor(() => {
            expect(screen.getByLabelText('IMAP Server')).toBeInTheDocument()
        })

        const imapServerInput = screen.getByLabelText('IMAP Server')
        await user.clear(imapServerInput)
        await user.type(imapServerInput, 'imap.newserver.com:993')

        expect(imapServerInput).toHaveValue('imap.newserver.com:993')
    })

    it('should submit settings form', async () => {
        vi.mocked(apiModule.api.getSettings).mockRejectedValue(new Error('Not found'))
        vi.mocked(apiModule.api.saveSettings).mockResolvedValue()

        const user = userEvent.setup()
        renderSettingsPage()

        await waitFor(() => {
            expect(screen.getByLabelText('IMAP Server')).toBeInTheDocument()
        })

        await user.type(screen.getByLabelText('IMAP Server'), 'imap.example.com:993')
        await user.type(screen.getByLabelText('IMAP Username'), 'user@example.com')
        await user.type(screen.getByLabelText('IMAP Password'), 'password123')
        await user.type(screen.getByLabelText('SMTP Server'), 'smtp.example.com:587')
        await user.type(screen.getByLabelText('SMTP Username'), 'user@example.com')
        await user.type(screen.getByLabelText('SMTP Password'), 'password123')

        const submitButton = screen.getByRole('button', {
            name: /Save Settings/i,
        })
        await user.click(submitButton)

        await waitFor(() => {
            expect(apiModule.api.saveSettings).toHaveBeenCalled()
        })
    })

    it('should show success message after saving', async () => {
        vi.mocked(apiModule.api.getSettings).mockResolvedValue(mockSettings)
        vi.mocked(apiModule.api.saveSettings).mockResolvedValue()

        const user = userEvent.setup()
        renderSettingsPage()

        await waitFor(() => {
            expect(screen.getByDisplayValue('imap.example.com:993')).toBeInTheDocument()
        })

        await user.type(screen.getByLabelText('IMAP Password'), 'test123')
        await user.type(screen.getByLabelText('SMTP Password'), 'test123')

        const submitButton = screen.getByRole('button', {
            name: /Save Settings/i,
        })
        await user.click(submitButton)

        await waitFor(
            () => {
                expect(screen.getByText('Settings saved successfully')).toBeInTheDocument()
            },
            { timeout: 5000 },
        )
    })

    it('should show error message on save failure', async () => {
        vi.mocked(apiModule.api.getSettings).mockResolvedValue(mockSettings)
        vi.mocked(apiModule.api.saveSettings).mockRejectedValue(new Error('Save failed'))

        const user = userEvent.setup()
        renderSettingsPage()

        await waitFor(() => {
            expect(screen.getByDisplayValue('imap.example.com:993')).toBeInTheDocument()
        })

        await user.type(screen.getByLabelText('IMAP Password'), 'test123')
        await user.type(screen.getByLabelText('SMTP Password'), 'test123')

        const submitButton = screen.getByRole('button', {
            name: /Save Settings/i,
        })
        await user.click(submitButton)

        await waitFor(
            () => {
                expect(screen.getByText('Error: Save failed')).toBeInTheDocument()
            },
            { timeout: 5000 },
        )
    })
})
