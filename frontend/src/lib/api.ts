const API_BASE_URL = '/api/v1'

/**
 * Encodes a thread ID (Message-ID with angle brackets) to URL-safe base64.
 * This makes the ID safe for URLs without percent-encoding.
 */
export function encodeThreadIdForUrl(threadId: string): string {
    // Convert string to base64 using TextEncoder for proper UTF-8 handling
    const utf8Bytes = new TextEncoder().encode(threadId)
    const base64 = btoa(String.fromCharCode(...utf8Bytes))
    // Make it URL-safe: replace + with -, / with _, and remove padding =
    return base64.replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '')
}

/**
 * Decodes a URL-safe base64 thread ID back to the original Message-ID.
 */
export function decodeThreadIdFromUrl(encoded: string): string {
    // Restore base64 characters: - to +, _ to /
    let base64 = encoded.replace(/-/g, '+').replace(/_/g, '/')
    // Add padding if needed (base64 strings should be multiple of 4)
    while (base64.length % 4) {
        base64 += '='
    }
    // Decode from base64
    try {
        const binaryString = atob(base64)
        const bytes = new Uint8Array(binaryString.length)
        for (let i = 0; i < binaryString.length; i++) {
            bytes[i] = binaryString.charCodeAt(i)
        }
        return new TextDecoder().decode(bytes)
    } catch (e) {
        const errorMessage = e instanceof Error ? e.message : String(e)
        throw new Error(`Failed to decode thread ID: ${errorMessage}`)
    }
}

export interface AuthStatus {
    isSetupComplete: boolean
}

export interface UserSettings {
    imap_server_hostname: string
    imap_username: string
    imap_password: string
    imap_password_set?: boolean
    smtp_server_hostname: string
    smtp_username: string
    smtp_password: string
    smtp_password_set?: boolean
    undo_send_delay_seconds: number
    pagination_threads_per_page: number
}

export interface Folder {
    name: string
    role: 'inbox' | 'sent' | 'drafts' | 'spam' | 'trash' | 'archive' | 'other'
}

export interface Message {
    id: string
    thread_id: string
    user_id: string
    imap_uid: number
    imap_folder_name: string
    message_id_header: string
    from_address: string
    to_addresses: string[]
    cc_addresses: string[]
    sent_at: string | null
    subject: string
    unsafe_body_html: string
    body_text: string
    is_read: boolean
    is_starred: boolean
    attachments?: Attachment[]
}

export interface Attachment {
    id: string
    message_id: string
    filename: string
    mime_type: string
    size_bytes: number
    is_inline: boolean
    content_id?: string
}

export interface Thread {
    id: string
    stable_thread_id: string
    subject: string
    user_id: string
    first_message_from_address?: string
    messages?: Message[]
}

export interface Pagination {
    total_count: number
    page: number
    per_page: number
}

export interface ThreadsResponse {
    threads: Thread[] | null
    pagination: Pagination
}

function getAuthHeaders() {
    return {
        Authorization: 'Bearer token',
    }
}

export const api = {
    async getAuthStatus(): Promise<AuthStatus> {
        const response = await fetch(`${API_BASE_URL}/auth/status`, {
            credentials: 'include',
            headers: getAuthHeaders(),
        })
        if (!response.ok) {
            throw new Error('Failed to fetch auth status')
        }
        return (await response.json()) as Promise<AuthStatus>
    },

    async getSettings(): Promise<UserSettings> {
        const response = await fetch(`${API_BASE_URL}/settings`, {
            credentials: 'include',
            headers: getAuthHeaders(),
        })
        if (!response.ok) {
            // Preserve status code information so caller can distinguish 404 from other errors
            const error = new Error('Failed to fetch settings')
            ;(error as Error & { status?: number }).status = response.status
            throw error
        }
        return (await response.json()) as Promise<UserSettings>
    },

    async saveSettings(settings: UserSettings): Promise<void> {
        const response = await fetch(`${API_BASE_URL}/settings`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                ...getAuthHeaders(),
            },
            credentials: 'include',
            body: JSON.stringify(settings),
        })
        if (!response.ok) {
            throw new Error('Failed to save settings')
        }
    },

    async getFolders(): Promise<Folder[]> {
        const response = await fetch(`${API_BASE_URL}/folders`, {
            credentials: 'include',
            headers: getAuthHeaders(),
        })
        if (!response.ok) {
            const errorText = await response.text()
            // Try to extract a meaningful error message from the response
            const errorMessage =
                errorText && errorText.length > 0 && errorText.length < 200
                    ? errorText
                    : 'Failed to fetch folders'
            throw new Error(errorMessage)
        }
        return (await response.json()) as Promise<Folder[]>
    },

    async getThreads(folder: string, page: number = 1, limit?: number): Promise<ThreadsResponse> {
        const params = new URLSearchParams({
            folder,
            page: page.toString(),
        })
        if (limit !== undefined) {
            params.append('limit', limit.toString())
        }
        const response = await fetch(`${API_BASE_URL}/threads?${params.toString()}`, {
            credentials: 'include',
            headers: getAuthHeaders(),
        })
        if (!response.ok) {
            throw new Error('Failed to fetch threads')
        }
        return (await response.json()) as Promise<ThreadsResponse>
    },

    async getThread(threadId: string): Promise<Thread> {
        // threadId is expected to be the raw Message-ID (with angle brackets)
        // We need to encode it for the URL path
        const encodedId = encodeURIComponent(threadId)
        const response = await fetch(`${API_BASE_URL}/thread/${encodedId}`, {
            credentials: 'include',
            headers: getAuthHeaders(),
        })
        if (!response.ok) {
            throw new Error('Failed to fetch thread')
        }
        return (await response.json()) as Promise<Thread>
    },

    async search(query: string, page: number = 1, limit?: number): Promise<ThreadsResponse> {
        const params = new URLSearchParams({
            q: query,
            page: page.toString(),
        })
        if (limit !== undefined) {
            params.append('limit', limit.toString())
        }
        const response = await fetch(`${API_BASE_URL}/search?${params.toString()}`, {
            credentials: 'include',
            headers: getAuthHeaders(),
        })
        if (!response.ok) {
            throw new Error('Failed to search')
        }
        return (await response.json()) as Promise<ThreadsResponse>
    },
}
