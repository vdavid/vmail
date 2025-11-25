import { useState } from 'react'
import * as React from 'react'
import { useNavigate } from 'react-router-dom'

import { validateSearchQuery } from '../lib/searchValidation'

interface HeaderProps {
    onToggleSidebar: () => void
}

export default function Header({ onToggleSidebar }: HeaderProps) {
    const [searchQuery, setSearchQuery] = useState('')
    const [validationError, setValidationError] = useState<string | null>(null)
    const navigate = useNavigate()

    const handleSearch = (e: React.FormEvent) => {
        e.preventDefault()
        const trimmedQuery = searchQuery.trim()

        // Validate query
        const error = validateSearchQuery(trimmedQuery)
        if (error) {
            setValidationError(error)
            return
        }

        setValidationError(null)
        void navigate(`/search?q=${encodeURIComponent(trimmedQuery)}`)
    }

    const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        setSearchQuery(e.target.value)
        // Clear validation error when user types
        if (validationError) {
            setValidationError(null)
        }
    }

    return (
        <header className='sticky top-0 z-30 border-b border-white/5 bg-slate-950/70 shadow-2xl shadow-black/50 backdrop-blur'>
            <div className='flex h-20 w-full items-center gap-4 px-4 sm:px-6 lg:px-8'>
                <button
                    type='button'
                    className='flex h-11 w-11 items-center justify-center rounded-full bg-slate-900 text-slate-300 ring-1 ring-white/10 transition hover:bg-slate-800 lg:hidden'
                    aria-label='Open navigation'
                    onClick={onToggleSidebar}
                >
                    ☰
                </button>
                <div className='flex items-center gap-3'>
                    <img
                        src='/vmail-logo-with-circle.svg'
                        alt='V-Mail logo'
                        className='h-10 w-10 rounded-full border border-white/10 bg-white/5 p-1'
                    />
                    <div className='hidden sm:block'>
                        <p className='text-base font-semibold text-white'>V-Mail</p>
                        <p className='text-xs text-slate-400'>A nice email UI without the G</p>
                    </div>
                </div>
                <form onSubmit={handleSearch} className='relative flex-1'>
                    <div className='relative'>
                        <span className='pointer-events-none absolute inset-y-0 left-4 flex items-center text-slate-400'>
                            ⌕
                        </span>
                        <input
                            type='text'
                            placeholder='Search mail...'
                            value={searchQuery}
                            onChange={handleInputChange}
                            className={`w-full rounded-full border px-6 py-3 pl-12 text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 ${
                                validationError
                                    ? 'border-red-500/70 bg-red-900/20 focus:border-red-400 focus:ring-red-400/40'
                                    : 'border-white/10 bg-slate-900/80 focus:border-blue-400 focus:ring-blue-400/40'
                            }`}
                            aria-label='Search mail'
                            aria-invalid={validationError !== null}
                            aria-describedby={validationError ? 'search-error' : undefined}
                        />
                        {validationError && (
                            <div
                                id='search-error'
                                className='absolute left-0 top-full mt-2 rounded-md bg-red-900/80 px-3 py-2 text-xs text-red-100 shadow-lg shadow-red-900/50'
                                role='alert'
                            >
                                {validationError}
                            </div>
                        )}
                    </div>
                </form>
                <div className='hidden items-center gap-3 lg:flex'>
                    <button
                        type='button'
                        className='rounded-full border border-white/10 bg-slate-900/70 px-4 py-2 text-sm text-slate-200 hover:border-white/30'
                        aria-label='Support'
                    >
                        Help
                    </button>
                    <button
                        type='button'
                        onClick={() => {
                            void navigate('/settings')
                        }}
                        className='rounded-full border border-white/10 bg-slate-900/70 px-4 py-2 text-sm text-slate-200 hover:border-white/30'
                        aria-label='Settings'
                    >
                        ⚙︎
                    </button>
                    <div className='flex h-11 w-11 items-center justify-center rounded-full bg-gradient-to-br from-blue-500 to-purple-500 text-sm font-semibold text-white'>
                        DV
                    </div>
                </div>
            </div>
        </header>
    )
}
