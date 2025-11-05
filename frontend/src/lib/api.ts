const API_BASE_URL = '/api/v1'

export interface AuthStatus {
    isAuthenticated: boolean
    isSetupComplete: boolean
}

export interface UserSettings {
    imap_server_hostname: string
    imap_username: string
    imap_password: string
    smtp_server_hostname: string
    smtp_username: string
    smtp_password: string
    archive_folder_name: string
    sent_folder_name: string
    drafts_folder_name: string
    trash_folder_name: string
    spam_folder_name: string
    undo_send_delay_seconds: number
    pagination_threads_per_page: number
}

export const api = {
    async getAuthStatus(): Promise<AuthStatus> {
        const response = await fetch(`${API_BASE_URL}/auth/status`, {
            credentials: 'include',
            headers: {
                'Authorization': 'Bearer token'
            },
        })
        if (!response.ok) {
            throw new Error('Failed to fetch auth status')
        }
        return response.json()
    },

    async getSettings(): Promise<UserSettings> {
        const response = await fetch(`${API_BASE_URL}/settings`, {
            credentials: 'include',
            headers: {
                'Authorization': 'Bearer token'
            },
        })
        if (!response.ok) {
            throw new Error('Failed to fetch settings')
        }
        return response.json()
    },

    async saveSettings(settings: UserSettings): Promise<void> {
        const response = await fetch(`${API_BASE_URL}/settings`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': 'Bearer token'
            },
            credentials: 'include',
            body: JSON.stringify(settings),
        })
        if (!response.ok) {
            throw new Error('Failed to save settings')
        }
    },
}
