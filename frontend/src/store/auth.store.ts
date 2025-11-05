import { create } from 'zustand'

interface AuthState {
    isSetupComplete: boolean
    setIsSetupComplete: (isComplete: boolean) => void
}

export const useAuthStore = create<AuthState>((set) => ({
    isSetupComplete: false,
    setIsSetupComplete: (isComplete: boolean) => set({ isSetupComplete: isComplete }),
}))
