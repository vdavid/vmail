import { useState } from 'react'
import * as React from 'react'
import { useNavigate } from 'react-router-dom'

import { validateSearchQuery } from '../lib/searchValidation'

export default function Header() {
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
        <header className='border-b border-gray-200 bg-white'>
            <div className='flex h-16 items-center px-6'>
                <form onSubmit={handleSearch} className='flex-1'>
                    <div className='relative'>
                        <input
                            type='text'
                            placeholder='Search mail...'
                            value={searchQuery}
                            onChange={handleInputChange}
                            className={`w-full rounded-md border px-4 py-2 pl-10 text-sm focus:outline-none focus:ring-1 ${
                                validationError
                                    ? 'border-red-300 bg-red-50 focus:border-red-500 focus:ring-red-500'
                                    : 'border-gray-300 bg-gray-50 focus:border-blue-500 focus:ring-blue-500'
                            }`}
                            aria-label='Search mail'
                            aria-invalid={validationError !== null}
                            aria-describedby={validationError ? 'search-error' : undefined}
                        />
                        <div className='pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3'>
                            <span className='text-gray-400' aria-hidden='true'>
                                🔍
                            </span>
                        </div>
                        {validationError && (
                            <div
                                id='search-error'
                                className='absolute left-0 top-full mt-1 rounded-md bg-red-100 px-2 py-1 text-xs text-red-700'
                                role='alert'
                            >
                                {validationError}
                            </div>
                        )}
                    </div>
                </form>
            </div>
        </header>
    )
}
