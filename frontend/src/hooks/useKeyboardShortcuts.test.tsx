import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { act } from 'react'
import { BrowserRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useKeyboardShortcuts } from './useKeyboardShortcuts'
import { useUIStore } from '../store/ui.store'
import * as api from '../lib/api'
import * as React from 'react'

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

        const { unmount } = renderHook(() => useKeyboardShortcuts(), {
            wrapper: createWrapper(),
        })

        expect(addEventListenerSpy).toHaveBeenCalledWith('keydown', expect.any(Function))

        unmount()

        expect(removeEventListenerSpy).toHaveBeenCalledWith('keydown', expect.any(Function))
    })

    it('increments selected index when "j" is pressed', async () => {
        const mockThreads = [
            { id: '1', stable_thread_id: 'thread-1', subject: 'Test 1', user_id: 'user-1' },
            { id: '2', stable_thread_id: 'thread-2', subject: 'Test 2', user_id: 'user-1' },
        ]
        vi.mocked(api.api.getThreads).mockResolvedValue(mockThreads)

        const queryClient = new QueryClient({
            defaultOptions: {
                queries: { retry: false },
            },
        })

        renderHook(() => useKeyboardShortcuts(), {
            wrapper: createWrapper(queryClient),
        })

        // Wait for the query to resolve by checking the query cache
        await waitFor(() => {
            const data = queryClient.getQueryData(['threads', 'INBOX'])
            expect(data).toEqual(mockThreads)
        })

        await act(async () => {
            const event = new KeyboardEvent('keydown', { key: 'j' })
            window.dispatchEvent(event)
        })

        await waitFor(() => {
            expect(useUIStore.getState().selectedThreadIndex).toBe(0)
        })
    })

    it('increments selected index when ArrowDown is pressed', async () => {
        const mockThreads = [
            { id: '1', stable_thread_id: 'thread-1', subject: 'Test 1', user_id: 'user-1' },
        ]
        vi.mocked(api.api.getThreads).mockResolvedValue(mockThreads)

        const queryClient = new QueryClient({
            defaultOptions: {
                queries: { retry: false },
            },
        })

        renderHook(() => useKeyboardShortcuts(), {
            wrapper: createWrapper(queryClient),
        })

        // Wait for the query to resolve
        await waitFor(() => {
            const data = queryClient.getQueryData(['threads', 'INBOX'])
            expect(data).toEqual(mockThreads)
        })

        await act(async () => {
            const event = new KeyboardEvent('keydown', { key: 'ArrowDown' })
            window.dispatchEvent(event)
        })

        await waitFor(() => {
            expect(useUIStore.getState().selectedThreadIndex).toBe(0)
        })
    })

    it('decrements selected index when "k" is pressed', () => {
        useUIStore.setState({ selectedThreadIndex: 1 })

        renderHook(() => useKeyboardShortcuts(), {
            wrapper: createWrapper(),
        })

        const event = new KeyboardEvent('keydown', { key: 'k' })
        window.dispatchEvent(event)

        expect(useUIStore.getState().selectedThreadIndex).toBe(0)
    })

    it('decrements selected index when ArrowUp is pressed', () => {
        useUIStore.setState({ selectedThreadIndex: 1 })

        renderHook(() => useKeyboardShortcuts(), {
            wrapper: createWrapper(),
        })

        const event = new KeyboardEvent('keydown', { key: 'ArrowUp' })
        window.dispatchEvent(event)

        expect(useUIStore.getState().selectedThreadIndex).toBe(0)
    })

    it('navigates to thread when "o" is pressed with selected thread', async () => {
        const mockThreads = [
            { id: '1', stable_thread_id: 'thread-1', subject: 'Test 1', user_id: 'user-1' },
        ]
        vi.mocked(api.api.getThreads).mockResolvedValue(mockThreads)

        const queryClient = new QueryClient({
            defaultOptions: {
                queries: { retry: false },
            },
        })

        renderHook(() => useKeyboardShortcuts(), {
            wrapper: createWrapper(queryClient),
        })

        // Wait for the query to resolve
        await waitFor(() => {
            const data = queryClient.getQueryData(['threads', 'INBOX'])
            expect(data).toEqual(mockThreads)
        })

        // First select a thread
        await act(async () => {
            const downEvent = new KeyboardEvent('keydown', { key: 'j' })
            window.dispatchEvent(downEvent)
        })

        await waitFor(() => {
            expect(useUIStore.getState().selectedThreadIndex).toBe(0)
        })

        // Then open it
        await act(async () => {
            const openEvent = new KeyboardEvent('keydown', { key: 'o' })
            window.dispatchEvent(openEvent)
        })

        // Wait for navigation
        await waitFor(() => {
            expect(mockNavigate).toHaveBeenCalledWith('/thread/thread-1')
        })
    })

    it('navigates to thread when Enter is pressed with selected thread', async () => {
        const mockThreads = [
            { id: '1', stable_thread_id: 'thread-1', subject: 'Test 1', user_id: 'user-1' },
        ]
        vi.mocked(api.api.getThreads).mockResolvedValue(mockThreads)

        const queryClient = new QueryClient({
            defaultOptions: {
                queries: { retry: false },
            },
        })

        renderHook(() => useKeyboardShortcuts(), {
            wrapper: createWrapper(queryClient),
        })

        // Wait for the query to resolve
        await waitFor(() => {
            const data = queryClient.getQueryData(['threads', 'INBOX'])
            expect(data).toEqual(mockThreads)
        })

        // First select a thread
        await act(async () => {
            const downEvent = new KeyboardEvent('keydown', { key: 'j' })
            window.dispatchEvent(downEvent)
        })

        await waitFor(() => {
            expect(useUIStore.getState().selectedThreadIndex).toBe(0)
        })

        // Then open it
        await act(async () => {
            const enterEvent = new KeyboardEvent('keydown', { key: 'Enter' })
            window.dispatchEvent(enterEvent)
        })

        // Wait for navigation
        await waitFor(() => {
            expect(mockNavigate).toHaveBeenCalledWith('/thread/thread-1')
        })
    })

    it('does not handle shortcuts when typing in input fields', () => {
        renderHook(() => useKeyboardShortcuts(), {
            wrapper: createWrapper(),
        })

        const input = document.createElement('input')
        document.body.appendChild(input)
        input.focus()

        const event = new KeyboardEvent('keydown', { key: 'j' })
        window.dispatchEvent(event)

        expect(useUIStore.getState().selectedThreadIndex).toBeNull()
    })
})
