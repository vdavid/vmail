import { describe, it, expect, beforeEach } from 'vitest'
import { useAuthStore } from './auth.store'

describe('useAuthStore', () => {
    beforeEach(() => {
        useAuthStore.setState({ isSetupComplete: false })
    })

    it('should initialize with isSetupComplete as false', () => {
        const { isSetupComplete } = useAuthStore.getState()
        expect(isSetupComplete).toBe(false)
    })

    it('should update isSetupComplete when setIsSetupComplete is called', () => {
        const { setIsSetupComplete } = useAuthStore.getState()

        setIsSetupComplete(true)
        expect(useAuthStore.getState().isSetupComplete).toBe(true)

        setIsSetupComplete(false)
        expect(useAuthStore.getState().isSetupComplete).toBe(false)
    })
})
