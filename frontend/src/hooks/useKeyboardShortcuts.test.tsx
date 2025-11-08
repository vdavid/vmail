import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import * as React from 'react'
import { act } from 'react'
import { BrowserRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import * as api from '../lib/api'
import { useUIStore } from '../store/ui.store'

import { useKeyboardShortcuts } from './useKeyboardShortcuts'

// Mock react-router-dom
const mockNavigate = vi.fn()
vi.mock('react-router-dom', async () => {
    const actual = await vi.importActual('react-router-dom')
    // noinspection JSUnusedGlobalSymbols
    return {
        ...actual,
        useNavigate: () => mockNavigate,
        useLocation: () => ({ pathname: '/', search: '' }),
        useSearchParams: () => [new URLSearchParams('folder=INBOX')],
    }
})

// Mock the API
vi.mock('../lib/api', () => ({
    api: {
        getThreads: vi.fn(),
    },
}))

const createWrapper = (queryClient?: QueryClient) => {
    const client =
        queryClient ||
        new QueryClient({
            defaultOptions: {
                queries: { retry: false },
            },
        })
    return ({ children }: { children: React.ReactNode }) => (
        <QueryClientProvider client={client}>
            <BrowserRouter>{children}</BrowserRouter>
        </QueryClientProvider>
    )
}

type MockThread = {
    id: string
    stable_thread_id: string
    subject: string
    user_id: string
}

const setupMockThreadsAndQueryClient = (mockThreads: MockThread[]) => {
    // eslint-disable-next-line @typescript-eslint/unbound-method
    vi.mocked(api.api.getThreads).mockResolvedValue(mockThreads)

    return new QueryClient({
        defaultOptions: {
            queries: { retry: false },
        },
    })
}

const waitForQueryToResolve = async (queryClient: QueryClient, mockThreads: MockThread[]) => {
    await waitFor(() => {
        const data = queryClient.getQueryData(['threads', 'INBOX'])
        expect(data).toEqual(mockThreads)
    })
}

const dispatchKeydown = (key: string) => {
    act(() => {
        const event = new KeyboardEvent('keydown', { key })
        window.dispatchEvent(event)
    })
}

const selectThread = async () => {
    dispatchKeydown('j')
    await waitFor(() => {
        expect(useUIStore.getState().selectedThreadIndex).toBe(0)
    })
}

describe('useKeyboardShortcuts', () => {
    beforeEach(() => {
        vi.clearAllMocks()
        useUIStore.setState({
            selectedThreadIndex: null,
        })
    })

    afterEach(() => {
        vi.clearAllMocks()
    })

    it('adds and removes event listeners on mount/unmount', () => {
        const addEventListenerSpy = vi.spyOn(window, 'addEventListener')
        const removeEventListenerSpy = vi.spyOn(window, 'removeEventListener')

        const { unmount } = renderHook(
            () => {
                useKeyboardShortcuts()
            },
            {
                wrapper: createWrapper(),
            },
        )

        expect(addEventListenerSpy).toHaveBeenCalledWith('keydown', expect.any(Function))

        unmount()

        expect(removeEventListenerSpy).toHaveBeenCalledWith('keydown', expect.any(Function))
    })

    it('increments selected index when "j" is pressed', async () => {
        const mockThreads = [
            { id: '1', stable_thread_id: 'thread-1', subject: 'Test 1', user_id: 'user-1' },
            { id: '2', stable_thread_id: 'thread-2', subject: 'Test 2', user_id: 'user-1' },
        ]
        const queryClient = setupMockThreadsAndQueryClient(mockThreads)

        renderHook(
            () => {
                useKeyboardShortcuts()
            },
            {
                wrapper: createWrapper(queryClient),
            },
        )

        await waitForQueryToResolve(queryClient, mockThreads)

        dispatchKeydown('j')

        await waitFor(() => {
            expect(useUIStore.getState().selectedThreadIndex).toBe(0)
        })
    })

    it('increments selected index when ArrowDown is pressed', async () => {
        const mockThreads = [
            { id: '1', stable_thread_id: 'thread-1', subject: 'Test 1', user_id: 'user-1' },
        ]
        const queryClient = setupMockThreadsAndQueryClient(mockThreads)

        renderHook(
            () => {
                useKeyboardShortcuts()
            },
            {
                wrapper: createWrapper(queryClient),
            },
        )

        await waitForQueryToResolve(queryClient, mockThreads)

        dispatchKeydown('ArrowDown')

        await waitFor(() => {
            expect(useUIStore.getState().selectedThreadIndex).toBe(0)
        })
    })

    it('decrements selected index when "k" is pressed', () => {
        useUIStore.setState({ selectedThreadIndex: 1 })

        renderHook(
            () => {
                useKeyboardShortcuts()
            },
            {
                wrapper: createWrapper(),
            },
        )

        dispatchKeydown('k')

        expect(useUIStore.getState().selectedThreadIndex).toBe(0)
    })

    it('decrements selected index when ArrowUp is pressed', () => {
        useUIStore.setState({ selectedThreadIndex: 1 })

        renderHook(
            () => {
                useKeyboardShortcuts()
            },
            {
                wrapper: createWrapper(),
            },
        )

        dispatchKeydown('ArrowUp')

        expect(useUIStore.getState().selectedThreadIndex).toBe(0)
    })

    it('navigates to thread when "o" is pressed with selected thread', async () => {
        const mockThreads = [
            { id: '1', stable_thread_id: 'thread-1', subject: 'Test 1', user_id: 'user-1' },
        ]
        const queryClient = setupMockThreadsAndQueryClient(mockThreads)

        renderHook(
            () => {
                useKeyboardShortcuts()
            },
            {
                wrapper: createWrapper(queryClient),
            },
        )

        await waitForQueryToResolve(queryClient, mockThreads)
        await selectThread()

        dispatchKeydown('o')

        await waitFor(() => {
            expect(mockNavigate).toHaveBeenCalledWith('/thread/thread-1')
        })
    })

    it('navigates to thread when Enter is pressed with selected thread', async () => {
        const mockThreads = [
            { id: '1', stable_thread_id: 'thread-1', subject: 'Test 1', user_id: 'user-1' },
        ]
        const queryClient = setupMockThreadsAndQueryClient(mockThreads)

        renderHook(
            () => {
                useKeyboardShortcuts()
            },
            {
                wrapper: createWrapper(queryClient),
            },
        )

        await waitForQueryToResolve(queryClient, mockThreads)
        await selectThread()

        dispatchKeydown('Enter')

        await waitFor(() => {
            expect(mockNavigate).toHaveBeenCalledWith('/thread/thread-1')
        })
    })

    it('does not handle shortcuts when typing in input fields', () => {
        renderHook(
            () => {
                useKeyboardShortcuts()
            },
            {
                wrapper: createWrapper(),
            },
        )

        const input = document.createElement('input')
        document.body.appendChild(input)
        input.focus()

        dispatchKeydown('j')

        expect(useUIStore.getState().selectedThreadIndex).toBeNull()
    })
})
