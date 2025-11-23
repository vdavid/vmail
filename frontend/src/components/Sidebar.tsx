import { useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'
import { Link, useLocation } from 'react-router-dom'

import { api, type Folder } from '../lib/api'

interface SidebarProps {
    isMobileOpen?: boolean
    onClose?: () => void
}

const ROLE_ORDER: Record<Folder['role'], number> = {
    inbox: 0,
    sent: 1,
    drafts: 2,
    spam: 3,
    trash: 4,
    archive: 5,
    other: 6,
}

export default function Sidebar({ isMobileOpen = false, onClose }: SidebarProps) {
    const location = useLocation()
    const {
        data: folders,
        isLoading,
        error,
        refetch,
        isRefetching,
    } = useQuery({
        queryKey: ['folders'],
        queryFn: () => api.getFolders(),
    })

    const sortedFolders = useMemo(() => {
        if (!folders) {
            return []
        }
        return [...folders].sort((a, b) => {
            const roleComparison = ROLE_ORDER[a.role] - ROLE_ORDER[b.role]
            if (roleComparison !== 0) {
                return roleComparison
            }
            return a.name.localeCompare(b.name)
        })
    }, [folders])

    const sidebarContent = (
        <div className='flex h-full flex-col'>
            <div className='flex flex-col gap-4 border-b border-white/10 pb-6'>
                <div className='flex items-center justify-between'>
                    <p className='text-base font-semibold text-white'>V-Mail</p>
                </div>
                <button
                    type='button'
                    className='flex w-full items-center justify-center gap-2 rounded-2xl bg-blue-500/90 px-4 py-3 text-sm font-semibold text-white shadow-lg shadow-blue-600/40 transition hover:bg-blue-500 disabled:opacity-60'
                    disabled
                    title='Compose is coming soon'
                >
                    ＋ Compose
                </button>
            </div>
            <nav className='mt-4 flex-1 space-y-1' aria-label='Sidebar navigation'>
                {isLoading ? (
                    <div className='px-2 py-2 text-sm text-slate-400'>Loading...</div>
                ) : error ? (
                    <div className='rounded-xl border border-red-400/30 bg-red-900/20 px-3 py-2 text-sm text-red-200'>
                        <p className='mb-2'>{error.message}</p>
                        <button
                            onClick={() => {
                                void refetch()
                            }}
                            disabled={isRefetching}
                            className='rounded-full bg-red-500/80 px-3 py-1 text-xs font-semibold text-white hover:bg-red-500 disabled:opacity-30'
                        >
                            {isRefetching ? 'Retrying...' : 'Retry'}
                        </button>
                    </div>
                ) : (
                    sortedFolders.map((folder) => {
                        const isInbox = folder.role === 'inbox'
                        const linkTo = isInbox ? '/' : `/?folder=${encodeURIComponent(folder.name)}`
                        const isActive = isInbox
                            ? location.pathname === '/' &&
                              (location.search === '' ||
                                  location.search.includes(
                                      `folder=${encodeURIComponent(folder.name)}`,
                                  ))
                            : location.pathname === '/' &&
                              location.search.includes(`folder=${encodeURIComponent(folder.name)}`)

                        return (
                            <Link
                                key={folder.name}
                                to={linkTo}
                                className={`group flex items-center gap-3 rounded-2xl px-3 py-2 text-sm font-medium transition ${
                                    isActive
                                        ? 'bg-white/10 text-white shadow-inner shadow-white/10'
                                        : 'text-slate-300 hover:bg-white/5 hover:text-white'
                                }`}
                                aria-current={isActive ? 'page' : undefined}
                                onClick={() => {
                                    onClose?.()
                                }}
                            >
                                <span
                                    className={`flex h-7 w-7 items-center justify-center rounded-full text-xs ${
                                        isActive
                                            ? 'bg-white/20 text-white'
                                            : 'bg-white/5 text-slate-300'
                                    }`}
                                    aria-hidden='true'
                                >
                                    {folder.name.slice(0, 2).toUpperCase()}
                                </span>
                                <span className='truncate'>{folder.name}</span>
                            </Link>
                        )
                    })
                )}
            </nav>
            <div className='mt-4 border-t border-white/10 pt-4'>
                <Link
                    to='/settings'
                    className='flex items-center gap-3 text-sm text-slate-300 transition hover:text-white'
                    onClick={() => {
                        onClose?.()
                    }}
                >
                    <span className='text-base' aria-hidden='true'>
                        ⚙︎
                    </span>
                    Settings
                </Link>
            </div>
        </div>
    )

    return (
        <>
            <aside className='hidden w-64 flex-shrink-0 rounded-3xl bg-slate-950/70 p-6 shadow-2xl shadow-black/40 backdrop-blur lg:flex'>
                {sidebarContent}
            </aside>

            <div
                className={`fixed inset-0 z-40 bg-black/60 backdrop-blur-sm transition-opacity duration-300 lg:hidden ${
                    isMobileOpen ? 'opacity-100' : 'pointer-events-none opacity-0'
                }`}
                aria-hidden={!isMobileOpen}
                onClick={onClose}
            />
            <aside
                className={`fixed inset-y-0 left-0 z-50 w-72 transform bg-slate-950/80 p-6 shadow-2xl shadow-black/60 backdrop-blur transition-transform duration-300 lg:hidden ${
                    isMobileOpen ? 'translate-x-0' : '-translate-x-full'
                }`}
                aria-hidden={!isMobileOpen}
            >
                <div className='mb-4 flex items-center justify-between'>
                    <p className='text-base font-semibold text-white'>Menu</p>
                    <button
                        type='button'
                        className='rounded-full border border-white/10 px-3 py-1 text-xs text-slate-200'
                        onClick={onClose}
                        aria-label='Close navigation'
                    >
                        Close
                    </button>
                </div>
                {sidebarContent}
            </aside>
        </>
    )
}
