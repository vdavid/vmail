import { useQuery } from '@tanstack/react-query'
import { useEffect } from 'react'
import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'

import { api, type AuthStatus } from '../lib/api'
import { useAuthStore } from '../store/auth.store'

interface AuthWrapperProps {
    children: ReactNode
}

export default function AuthWrapper({ children }: AuthWrapperProps) {
    const location = useLocation()
    const { isSetupComplete, setIsSetupComplete } = useAuthStore()

    const { data, isLoading, isError } = useQuery<AuthStatus>({
        queryKey: ['authStatus'],
        queryFn: () => api.getAuthStatus(),
        retry: false,
        refetchOnWindowFocus: false,
    })

    useEffect(() => {
        if (data) {
            setIsSetupComplete(data.isSetupComplete)
        } else if (isError) {
            setIsSetupComplete(false)
        }
    }, [data, isError, setIsSetupComplete])

    if (isLoading) {
        return (
            <div className='flex h-screen items-center justify-center bg-gray-50'>
                <div className='text-center'>
                    <div className='mb-4 text-4xl'>📧</div>
                    <div className='text-lg text-gray-600'>Loading V-Mail...</div>
                </div>
            </div>
        )
    }

    if (!isSetupComplete && location.pathname !== '/settings') {
        return <Navigate to='/settings' replace />
    }

    return <>{children}</>
}
