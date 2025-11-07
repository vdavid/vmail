import { Link, useLocation } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'

export default function Sidebar() {
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

    return (
        <div className='flex h-full w-64 flex-col border-r border-gray-200 bg-white'>
            <div className='flex h-16 items-center border-b border-gray-200 px-6'>
                <h1 className='text-xl font-bold text-gray-900'>V-Mail</h1>
            </div>
            <nav className='flex-1 space-y-1 px-3 py-4' aria-label='Sidebar'>
                {isLoading ? (
                    <div className='px-3 py-2 text-sm text-gray-500'>Loading...</div>
                ) : error ? (
                    <div className='px-3 py-2'>
                        <div className='mb-2 text-sm text-red-600'>
                            {error.message}
                        </div>
                        <button
                            onClick={() => refetch()}
                            disabled={isRefetching}
                            className='rounded-md bg-blue-600 px-3 py-1.5 text-xs text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50'
                        >
                            {isRefetching ? 'Retrying...' : 'Retry'}
                        </button>
                    </div>
                ) : (
                    folders?.map((folder) => {
                        const folderParam = `folder=${encodeURIComponent(folder.name)}`
                        const isActive =
                            location.pathname === '/' && location.search.includes(folderParam)
                        return (
                            <Link
                                key={folder.name}
                                to={`/?${folderParam}`}
                                className={`group flex items-center rounded-md px-3 py-2 text-sm font-medium ${
                                    isActive
                                        ? 'bg-gray-100 text-gray-900'
                                        : 'text-gray-700 hover:bg-gray-50 hover:text-gray-900'
                                }`}
                                aria-current={isActive ? 'page' : undefined}
                            >
                                {folder.name}
                            </Link>
                        )
                    })
                )}
            </nav>
            <div className='border-t border-gray-200 p-4'>
                <Link
                    to='/settings'
                    className='flex items-center text-sm text-gray-700 hover:text-gray-900'
                >
                    <span className='mr-3 text-lg' aria-hidden='true'>
                        ⚙️
                    </span>
                    Settings
                </Link>
            </div>
        </div>
    )
}
