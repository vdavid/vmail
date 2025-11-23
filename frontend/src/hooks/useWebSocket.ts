import { useQueryClient } from '@tanstack/react-query'
import { useEffect, useRef } from 'react'

import { useConnectionStore } from '../store/connection.store'

export function useWebSocket() {
    const queryClient = useQueryClient()
    const { setStatus, setLastError, forceReconnectToken } = useConnectionStore()
    const queryClientRef = useRef(queryClient)
    const socketRef = useRef<WebSocket | null>(null)
    const socketCreationTimeRef = useRef<number | null>(null)

    // Keep refs up to date without causing re-renders
    useEffect(() => {
        queryClientRef.current = queryClient
    }, [queryClient])

    useEffect(() => {
        // Check if we already have an active socket from a previous effect run (StrictMode)
        const existingSocket = socketRef.current
        if (existingSocket) {
            const state = existingSocket.readyState
            // Try to reuse existing socket
            if (state === WebSocket.OPEN || state === WebSocket.CONNECTING) {
                // Update status to match the current state
                if (state === WebSocket.OPEN) {
                    setStatus('connected')
                    setLastError(null)
                } else {
                    setStatus('connecting')
                }
                // Return early - don't create a new socket or set up handlers
                // The existing socket will continue with its existing handlers
                // No cleanup needed since we're not creating anything new
                return
            }
            // Socket is closed or closing, create a new one
        }

        setStatus('connecting')

        // Get the token (currently hardcoded as "token", same as used in API calls).
        // TODO: When Authelia is implemented, this should get the actual JWT token.
        const token = 'token'

        const wsEnvUrl = import.meta.env.VITE_WS_URL as string | undefined
        let wsUrl: string
        if (wsEnvUrl && wsEnvUrl.length > 0) {
            wsUrl = wsEnvUrl
        } else {
            const baseUrl = `${window.location.origin.replace(/^http/, 'ws')}/api/v1/ws`
            // Append token as query parameter since WebSocket connections can't set headers.
            wsUrl = `${baseUrl}?token=${encodeURIComponent(token)}`
        }

        // Connect
        const socket = new WebSocket(wsUrl)
        const socketInstance = socket
        socketRef.current = socket
        socketCreationTimeRef.current = Date.now()

        socket.onopen = () => {
            // Only update state if this is still the current socket
            if (socketRef.current === socketInstance) {
                setStatus('connected')
                setLastError(null)
            } else {
                // Connection opened but socket ref changed (StrictMode)
                socket.close()
            }
        }

        socket.onerror = (error) => {
            // eslint-disable-next-line no-console -- We do want to log this in production too
            console.error('WebSocket: Error occurred', error, 'readyState:', socket.readyState)
            if (socketRef.current === socketInstance) {
                setStatus('disconnected')
                setLastError('WebSocket error')
            }
        }

        socket.onclose = () => {
            if (socketRef.current === socketInstance) {
                socketRef.current = null
                setStatus('disconnected')
            }
        }

        socket.onmessage = (event) => {
            if (socketRef.current !== socketInstance) {
                return
            }
            try {
                const data = JSON.parse(event.data as string) as { type?: string; folder?: string }
                if (data.type === 'new_email' && data.folder) {
                    // Invalidate all queries that start with ['threads', folder]
                    // This will match ['threads', folder, page, limit] for any page/limit
                    queryClientRef.current
                        .invalidateQueries({
                            queryKey: ['threads', data.folder],
                            exact: false, // Match all queries that start with this key
                        })
                        .catch((err: unknown) => {
                            // eslint-disable-next-line no-console -- Weird error, better log it
                            console.error('WebSocket: Failed to invalidate queries', err)
                        })
                }
            } catch (err) {
                // eslint-disable-next-line no-console -- We actually want to log this
                console.error('WebSocket: Failed to parse message', err, event.data)
            }
        }

        return () => {
            // Cleanup method

            // Only clean up if this is still the current socket
            if (socketRef.current !== socketInstance) {
                return
            }

            const timeSinceCreation = socketCreationTimeRef.current
                ? Date.now() - socketCreationTimeRef.current
                : Infinity

            // In StrictMode, effects run twice. If cleanup is called very soon after creation,
            // it's likely a StrictMode double-mount. Don't close immediately.
            // The 100 ms is kinda arbitrary, so this is kinda a hack.
            if (timeSinceCreation < 100) {
                return
            }

            socketRef.current = null
            if (
                socket.readyState === WebSocket.CONNECTING ||
                socket.readyState === WebSocket.OPEN
            ) {
                // Close socket as cleanup
                socket.close()
            }
        }
    }, [forceReconnectToken, setLastError, setStatus])
}
