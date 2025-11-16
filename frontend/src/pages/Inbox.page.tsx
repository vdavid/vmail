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
            <div className='flex h-full flex-col gap-4 p-6 text-slate-200'>
                <div className='h-8 w-32 rounded-full bg-white/10' />
                <div className='rounded-3xl border border-white/5 bg-white/5 p-6 text-sm'>
                    Loading...
                </div>
            </div>
        )
    }

    if (error) {
        return (
            <div className='p-6 text-slate-100'>
                <h1 className='text-2xl font-bold'>{folder === 'INBOX' ? 'Inbox' : folder}</h1>
                <p className='mt-4 rounded-2xl bg-red-900/40 p-4 text-sm text-red-100'>
                    Error loading threads: {error.message}
                </p>
            </div>
        )
    }

    return (
        <div className='flex h-full flex-col'>
            <div className='border-b border-white/5 px-4 py-4 sm:px-6'>
                <div className='flex flex-wrap items-center gap-3'>
                    <h1 className='text-xl font-semibold text-white'>
                        {folder === 'INBOX' ? 'Inbox' : folder}
                    </h1>
                    <span className='rounded-full border border-white/10 px-3 py-1 text-xs uppercase tracking-wide text-slate-300'>
                        Compact view
                    </span>
                </div>
            </div>
            <div className='flex-1 overflow-y-auto divide-y divide-white/5'>
                {threadsResponse &&
                threadsResponse.threads &&
                threadsResponse.threads.length > 0 ? (
                    threadsResponse.threads.map((thread, index) => (
                        <EmailListItem
                            key={thread.id}
                            thread={thread}
                            isSelected={selectedThreadIndex === index}
                        />
                    ))
                ) : (
                    <div className='p-6 text-center text-sm text-slate-400'>No threads found</div>
                )}
            </div>
            {threadsResponse?.pagination && (
                <EmailListPagination pagination={threadsResponse.pagination} />
            )}
        </div>
    )
}
