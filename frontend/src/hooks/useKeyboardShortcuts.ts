import { useEffect } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useUIStore } from '../store/ui.store'
import { useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'
import { useSearchParams } from 'react-router-dom'

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

    // Get threads for navigation
    const { data: threads } = useQuery({
        queryKey: ['threads', folder],
        queryFn: () => api.getThreads(folder),
        enabled: location.pathname === '/',
    })

    useEffect(() => {
        const handleKeyDown = (event: KeyboardEvent) => {
            // Don't handle shortcuts when typing in input fields
            if (
                event.target instanceof HTMLInputElement ||
                event.target instanceof HTMLTextAreaElement
            ) {
                return
            }

            const isInbox = location.pathname === '/'
            const isThreadView = location.pathname.startsWith('/thread/')

            // j or ↓: Move to next item
            if (event.key === 'j' || event.key === 'ArrowDown') {
                event.preventDefault()
                if (isInbox && threads && threads.length > 0) {
                    incrementSelectedIndex(threads.length)
                }
            }

            // k or ↑: Move to previous item
            if (event.key === 'k' || event.key === 'ArrowUp') {
                event.preventDefault()
                if (isInbox) {
                    decrementSelectedIndex()
                }
            }

            // o or Enter: Open selected thread
            if ((event.key === 'o' || event.key === 'Enter') && isInbox) {
                event.preventDefault()
                if (selectedThreadIndex !== null && threads && threads[selectedThreadIndex]) {
                    navigate(`/thread/${threads[selectedThreadIndex].stable_thread_id}`)
                    setSelectedThreadIndex(null)
                }
            }

            // u: Go back to inbox from thread view
            if (event.key === 'u' && isThreadView) {
                event.preventDefault()
                navigate('/')
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
