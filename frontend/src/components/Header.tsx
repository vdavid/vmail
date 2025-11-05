import { useState } from 'react'

export default function Header() {
    const [searchQuery, setSearchQuery] = useState('')

    const handleSearch = (e: React.FormEvent) => {
        e.preventDefault()
        console.log('Search:', searchQuery)
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
                            onChange={(e) => setSearchQuery(e.target.value)}
                            className='w-full rounded-md border border-gray-300 bg-gray-50 px-4 py-2 pl-10 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500'
                            aria-label='Search mail'
                        />
                        <div className='pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3'>
                            <span className='text-gray-400' aria-hidden='true'>
                                🔍
                            </span>
                        </div>
                    </div>
                </form>
            </div>
        </header>
    )
}
