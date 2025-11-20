import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, render } from '@testing-library/react'
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'

import { useConnectionStore } from '../store/connection.store'

import { useWebSocket } from './useWebSocket'

class MockSocket {
    static instances: MockSocket[] = []
    onopen: (() => void) | null = null
    onmessage: ((event: MessageEvent) => void) | null = null
    onerror: (() => void) | null = null
    onclose: ((event: CloseEvent) => void) | null = null

    url: string

    constructor(url: string) {
        this.url = url
        MockSocket.instances.push(this)
    }

    send(): void {}

    close() {
        if (this.onclose) {
            this.onclose(new CloseEvent('close'))
        }
    }
}

function TestComponent() {
    useWebSocket()
    return null
}

function renderWithClient(queryClient: QueryClient) {
    return render(
        <QueryClientProvider client={queryClient}>
            <TestComponent />
        </QueryClientProvider>,
    )
}

describe('useWebSocket', () => {
    beforeEach(() => {
        // Reset connection store between tests
        useConnectionStore.setState({
            status: 'connecting',
            lastError: null,
            forceReconnectToken: 0,
        })
        vi.stubGlobal('WebSocket', MockSocket as unknown as typeof WebSocket)
    })

    afterEach(() => {
        MockSocket.instances = []
        vi.unstubAllGlobals()
    })

    it('invalidates threads query when new_email message is received', () => {
        const queryClient = new QueryClient({
            defaultOptions: {
                queries: { retry: false },
            },
        })
        const invalidateSpy = vi
            .spyOn(queryClient, 'invalidateQueries')
            .mockResolvedValue(undefined)

        renderWithClient(queryClient)

        // Grab the created mock socket instance.
        const socket = MockSocket.instances[0]
        expect(socket).toBeDefined()

        act(() => {
            const event = new MessageEvent('message', {
                data: JSON.stringify({ type: 'new_email', folder: 'INBOX' }),
            })
            socket.onmessage?.(event)
        })

        expect(invalidateSpy).toHaveBeenCalledWith({
            queryKey: ['threads', 'INBOX'],
            exact: false,
        })
    })
})
