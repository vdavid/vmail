import { useState, type ReactNode } from 'react'

import { useKeyboardShortcuts } from '../hooks/useKeyboardShortcuts'

import Header from './Header'
import Sidebar from './Sidebar'

interface LayoutProps {
    children: ReactNode
}

export default function Layout({ children }: LayoutProps) {
    useKeyboardShortcuts()
    const [isSidebarOpen, setIsSidebarOpen] = useState(false)

    return (
        <div className='min-h-screen bg-transparent text-slate-100'>
            <Header
                onToggleSidebar={() => {
                    setIsSidebarOpen(true)
                }}
            />
            <div className='flex w-full flex-1 gap-6 px-4 pb-12 pt-6 sm:px-6 lg:px-8'>
                <Sidebar
                    isMobileOpen={isSidebarOpen}
                    onClose={() => {
                        setIsSidebarOpen(false)
                    }}
                />
                <main className='flex-1 overflow-hidden rounded-3xl bg-slate-950/60 shadow-[0_35px_60px_-15px_rgba(0,0,0,0.75)] backdrop-blur'>
                    <div className='flex h-full flex-col'>{children}</div>
                </main>
            </div>
        </div>
    )
}
