import { useQuery } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'

import EmailListItem from '../components/EmailListItem'
import EmailListPagination from '../components/EmailListPagination'
import { api } from '../lib/api'
import { useUIStore } from '../store/ui.store'

export default function InboxPage() {
    const [searchParams] = useSearchParams()
    const folder = searchParams.get('folder') || 'INBOX'
    const page = parseInt(searchParams.get('page') || '1', 10)
    const selectedThreadIndex = useUIStore((state) => state.selectedThreadIndex)

    // Get user settings to determine pagination limit
    const { data: settings } = useQuery({
        queryKey: ['settings'],
        queryFn: () => api.getSettings(),
    })

    const limit = settings?.pagination_threads_per_page ?? 100

    const {
        data: threadsResponse,
        isLoading,
        error,
    } = useQuery({
        queryKey: ['threads', folder, page, limit],
        queryFn: () => api.getThreads(folder, page, limit),
        enabled: !!settings, // Wait for settings to load before fetching threads
    })

    if (isLoading) {
        return (
            <div className='p-6'>
                <h1 className='text-2xl font-bold text-gray-900'>
                    {folder === 'INBOX' ? 'Inbox' : folder}
                </h1>
                <p className='mt-4 text-gray-600'>Loading...</p>
            </div>
        )
    }

    if (error) {
        return (
            <div className='p-6'>
                <h1 className='text-2xl font-bold text-gray-900'>
                    {folder === 'INBOX' ? 'Inbox' : folder}
                </h1>
                <p className='mt-4 text-red-600'>Error loading threads: {error.message}</p>
            </div>
        )
    }

    return (
        <div className='flex h-full flex-col'>
            <div className='border-b border-gray-200 px-6 py-4'>
                <h1 className='text-2xl font-bold text-gray-900'>
                    {folder === 'INBOX' ? 'Inbox' : folder}
                </h1>
            </div>
            <div className='flex-1 overflow-y-auto'>
                {threadsResponse &&
                threadsResponse.threads &&
                threadsResponse.threads.length > 0 ? (
                    <div>
                        {threadsResponse.threads.map((thread, index) => (
                            <EmailListItem
                                key={thread.id}
                                thread={thread}
                                isSelected={selectedThreadIndex === index}
                            />
                        ))}
                    </div>
                ) : (
                    <div className='p-6 text-center text-gray-500'>No threads found</div>
                )}
            </div>
            {threadsResponse?.pagination && (
                <EmailListPagination pagination={threadsResponse.pagination} />
            )}
        </div>
    )
}
