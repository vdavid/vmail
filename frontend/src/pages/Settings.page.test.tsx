import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'

import * as apiModule from '../lib/api'
import type { UserSettings } from '../lib/api'

import SettingsPage from './Settings.page'

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
            <MemoryRouter>
                <QueryClientProvider client={queryClient}>
                    <SettingsPage />
                </QueryClientProvider>
            </MemoryRouter>,
        )
    }

    it('should show loading state initially', () => {
        // eslint-disable-next-line @typescript-eslint/unbound-method
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
        // eslint-disable-next-line @typescript-eslint/unbound-method
        vi.mocked(apiModule.api.getSettings).mockRejectedValue(new Error('Not found'))

        renderSettingsPage()

        await waitFor(() => {
            expect(screen.getByText('Settings')).toBeInTheDocument()
        })

        expect(screen.getByLabelText('IMAP server')).toBeInTheDocument()
        expect(screen.getByLabelText('IMAP username')).toBeInTheDocument()
        expect(screen.getByLabelText('SMTP server')).toBeInTheDocument()
    })

    it('should load existing settings', async () => {
        // eslint-disable-next-line @typescript-eslint/unbound-method
        vi.mocked(apiModule.api.getSettings).mockResolvedValue(mockSettings)

        renderSettingsPage()

        await waitFor(() => {
            expect(screen.getByDisplayValue('imap.example.com:993')).toBeInTheDocument()
        })

        const userFields = screen.getAllByDisplayValue('user@example.com')
        expect(userFields).toHaveLength(2)
    })

    it('should handle form input changes', async () => {
        // eslint-disable-next-line @typescript-eslint/unbound-method
        vi.mocked(apiModule.api.getSettings).mockRejectedValue(new Error('Not found'))

        const user = userEvent.setup()
        renderSettingsPage()

        await waitFor(() => {
            expect(screen.getByLabelText('IMAP server')).toBeInTheDocument()
        })

        const imapServerInput = screen.getByLabelText('IMAP server')
        await user.clear(imapServerInput)
        await user.type(imapServerInput, 'imap.new-server.com:993')

        expect(imapServerInput).toHaveValue('imap.new-server.com:993')
    })

    it('should submit settings form', async () => {
        // eslint-disable-next-line @typescript-eslint/unbound-method
        vi.mocked(apiModule.api.getSettings).mockRejectedValue(new Error('Not found'))
        // eslint-disable-next-line @typescript-eslint/unbound-method
        vi.mocked(apiModule.api.saveSettings).mockResolvedValue()

        const user = userEvent.setup()
        renderSettingsPage()

        await waitFor(() => {
            expect(screen.getByLabelText('IMAP server')).toBeInTheDocument()
        })

        await user.type(screen.getByLabelText('IMAP server'), 'imap.example.com:993')
        await user.type(screen.getByLabelText('IMAP username'), 'user@example.com')
        await user.type(screen.getByLabelText('IMAP password'), 'password123')
        await user.type(screen.getByLabelText('SMTP server'), 'smtp.example.com:587')
        await user.type(screen.getByLabelText('SMTP username'), 'user@example.com')
        await user.type(screen.getByLabelText('SMTP password'), 'password123')

        const submitButton = screen.getByRole('button', {
            name: /Save Settings/i,
        })
        await user.click(submitButton)

        await waitFor(() => {
            // eslint-disable-next-line @typescript-eslint/unbound-method
            expect(apiModule.api.saveSettings).toHaveBeenCalled()
        })
    })

    it('should show success message after saving', async () => {
        // eslint-disable-next-line @typescript-eslint/unbound-method
        vi.mocked(apiModule.api.getSettings).mockResolvedValue(mockSettings)
        // eslint-disable-next-line @typescript-eslint/unbound-method
        vi.mocked(apiModule.api.saveSettings).mockResolvedValue()

        const user = userEvent.setup()
        renderSettingsPage()

        await waitFor(() => {
            expect(screen.getByDisplayValue('imap.example.com:993')).toBeInTheDocument()
        })

        await user.type(screen.getByLabelText('IMAP password'), 'test123')
        await user.type(screen.getByLabelText('SMTP password'), 'test123')

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
        // eslint-disable-next-line @typescript-eslint/unbound-method
        vi.mocked(apiModule.api.getSettings).mockResolvedValue(mockSettings)
        // eslint-disable-next-line @typescript-eslint/unbound-method
        vi.mocked(apiModule.api.saveSettings).mockRejectedValue(new Error('Save failed'))

        const user = userEvent.setup()
        renderSettingsPage()

        await waitFor(() => {
            expect(screen.getByDisplayValue('imap.example.com:993')).toBeInTheDocument()
        })

        await user.type(screen.getByLabelText('IMAP password'), 'test123')
        await user.type(screen.getByLabelText('SMTP password'), 'test123')

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
