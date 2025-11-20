import { create } from 'zustand'

type ConnectionStatus = 'connected' | 'connecting' | 'disconnected'

interface ConnectionState {
    status: ConnectionStatus
    lastError: string | null
    forceReconnectToken: number
    setStatus: (status: ConnectionStatus) => void
    setLastError: (message: string | null) => void
    triggerReconnect: () => void
}

export const useConnectionStore = create<ConnectionState>((set) => ({
    status: 'connecting',
    lastError: null,
    forceReconnectToken: 0,
    setStatus: (status) => {
        set({ status })
    },
    setLastError: (message) => {
        set({ lastError: message })
    },
    triggerReconnect: () => {
        set((state) => ({
            forceReconnectToken: state.forceReconnectToken + 1,
            status: 'connecting',
        }))
    },
}))
