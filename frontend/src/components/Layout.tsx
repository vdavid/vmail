import type { ReactNode } from 'react'
import Sidebar from './Sidebar'
import Header from './Header'
import { useKeyboardShortcuts } from '../hooks/useKeyboardShortcuts'

interface LayoutProps {
    children: ReactNode
}

export default function Layout({ children }: LayoutProps) {
    useKeyboardShortcuts()

    return (
        <div className='flex h-screen overflow-hidden bg-gray-50'>
            <Sidebar />
            <div className='flex flex-1 flex-col overflow-hidden'>
                <Header />
                <main className='flex-1 overflow-auto'>
                    <div className='h-full'>{children}</div>
                </main>
            </div>
        </div>
    )
}
