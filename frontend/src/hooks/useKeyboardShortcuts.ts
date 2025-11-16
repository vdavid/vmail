import { useQuery } from '@tanstack/react-query'
import { useEffect } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useSearchParams } from 'react-router-dom'

import { api, encodeThreadIdForUrl } from '../lib/api'
import { useUIStore } from '../store/ui.store'

export function useKeyboardShortcuts() {
    const navigate = useNavigate()
    const location = useLocation()
    const [searchParams] = useSearchParams()
    const folder = searchParams.get('folder') || 'INBOX'
    const {
        selectedThreadIndex,
        incrementSelectedIndex,
        decrementSelectedIndex,
        setSelectedThreadIndex,
    } = useUIStore()

    // Get user settings to determine pagination limit
    const { data: settings } = useQuery({
        queryKey: ['settings'],
        queryFn: () => api.getSettings(),
    })

    const limit = settings?.pagination_threads_per_page ?? 100

    // Get threads for navigation
    const { data: threadsResponse } = useQuery({
        queryKey: ['threads', folder, 1, limit],
        queryFn: () => api.getThreads(folder, 1, limit),
        enabled: location.pathname === '/' && !!settings,
    })

    const threads = threadsResponse?.threads ?? null

    useEffect(() => {
        const isInbox = location.pathname === '/'
        const isThreadView = location.pathname.startsWith('/thread/')

        const handleNextItem = (event: KeyboardEvent) => {
            event.preventDefault()
            event.stopPropagation()
            if (isInbox && threads && threads.length > 0) {
                incrementSelectedIndex(threads.length)
            }
        }

        const handlePreviousItem = (event: KeyboardEvent) => {
            event.preventDefault()
            event.stopPropagation()
            if (isInbox) {
                decrementSelectedIndex()
            }
        }

        const handleOpenThread = (event: KeyboardEvent) => {
            event.preventDefault()
            if (
                selectedThreadIndex !== null &&
                threads &&
                selectedThreadIndex >= 0 &&
                selectedThreadIndex < threads.length &&
                threads[selectedThreadIndex]
            ) {
                const selectedThread = threads[selectedThreadIndex]
                void navigate(`/thread/${encodeThreadIdForUrl(selectedThread.stable_thread_id)}`)
                setSelectedThreadIndex(null)
            }
        }

        const handleBackToInbox = (event: KeyboardEvent) => {
            event.preventDefault()
            void navigate('/')
        }

        const handleKeyDown = (event: KeyboardEvent) => {
            // Don't handle shortcuts when typing in input fields
            if (
                event.target instanceof HTMLInputElement ||
                event.target instanceof HTMLTextAreaElement
            ) {
                return
            }

            // j or ↓: Move to next item
            if (event.key === 'j' || event.key === 'ArrowDown') {
                handleNextItem(event)
                return
            }

            // k or ↑: Move to previous item
            if (event.key === 'k' || event.key === 'ArrowUp') {
                handlePreviousItem(event)
                return
            }

            // o or Enter: Open selected thread
            if ((event.key === 'o' || event.key === 'Enter') && isInbox) {
                handleOpenThread(event)
                return
            }

            // u: Go back to inbox from thread view
            if (event.key === 'u' && isThreadView) {
                handleBackToInbox(event)
            }
        }

        window.addEventListener('keydown', handleKeyDown)
        return () => {
            window.removeEventListener('keydown', handleKeyDown)
        }
    }, [
        navigate,
        location.pathname,
        selectedThreadIndex,
        threads,
        incrementSelectedIndex,
        decrementSelectedIndex,
        setSelectedThreadIndex,
    ])
}
