import { useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'

import EmailListItem from '../components/EmailListItem'
import EmailListPagination from '../components/EmailListPagination'
import { api, type ThreadsResponse } from '../lib/api'
import { useUIStore } from '../store/ui.store'

function SearchHeader({ hasQuery, trimmedQuery }: { hasQuery: boolean; trimmedQuery: string }) {
    return (
        <div className='border-b border-white/5 px-4 py-4 text-white sm:px-6'>
            <div className='flex flex-wrap items-center gap-3'>
                <h1 data-testid='search-page-heading' className='text-xl font-semibold'>
                    {hasQuery ? 'Search results' : 'Search'}
                </h1>
                {hasQuery && (
                    <span className='rounded-full border border-white/10 px-3 py-1 text-xs uppercase tracking-wide text-slate-300'>
                        "{trimmedQuery}"
                    </span>
                )}
            </div>
            <p className='mt-2 text-sm text-slate-400'>
                {hasQuery ? 'Here is what we found.' : 'Use the search box above to find emails.'}
            </p>
        </div>
    )
}

function SearchResults({
    threadsResponse,
    selectedThreadIndex,
    hasQuery,
    trimmedQuery,
}: {
    threadsResponse: ThreadsResponse | undefined
    selectedThreadIndex: number | null
    hasQuery: boolean
    trimmedQuery: string
}) {
    if (!threadsResponse?.threads || threadsResponse.threads.length === 0) {
        return (
            <div className='p-6 text-center text-sm text-slate-400'>
                {hasQuery
                    ? `No results found for "${trimmedQuery}"`
                    : 'Enter a search query to find emails'}
            </div>
        )
    }

    return (
        <>
            {threadsResponse.threads.map((thread, index) => (
                <EmailListItem
                    key={thread.id}
                    thread={thread}
                    isSelected={selectedThreadIndex === index}
                />
            ))}
        </>
    )
}

function LoadingState() {
    return (
        <div className='flex h-full flex-col gap-4 p-6 text-slate-200'>
            <div className='h-8 w-40 rounded-full bg-white/10' />
            <div className='rounded-3xl border border-white/5 bg-white/5 p-6 text-sm'>
                Loading...
            </div>
        </div>
    )
}

function ErrorState({ error }: { error: Error }) {
    return (
        <div className='p-6 text-slate-100'>
            <h1 className='text-2xl font-bold'>Search</h1>
            <p className='mt-4 rounded-2xl bg-red-900/40 p-4 text-sm text-red-100'>
                Error searching: {error.message}
            </p>
        </div>
    )
}

export default function SearchPage() {
    const [searchParams] = useSearchParams()
    const query = searchParams.get('q') || ''
    const trimmedQuery = useMemo(() => query.trim(), [query])
    const hasQuery = trimmedQuery.length > 0
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
        queryKey: ['search', query, page, limit],
        queryFn: () => api.search(query, page, limit),
        enabled: !!settings, // Wait for settings (empty query is allowed)
    })

    if (isLoading) {
        return <LoadingState />
    }

    if (error) {
        return <ErrorState error={error} />
    }

    return (
        <div className='flex h-full flex-col'>
            <SearchHeader hasQuery={hasQuery} trimmedQuery={trimmedQuery} />
            <div className='flex-1 overflow-y-auto divide-y divide-white/5'>
                <SearchResults
                    threadsResponse={threadsResponse}
                    selectedThreadIndex={selectedThreadIndex}
                    hasQuery={hasQuery}
                    trimmedQuery={trimmedQuery}
                />
            </div>
            {threadsResponse?.pagination && (
                <EmailListPagination pagination={threadsResponse.pagination} />
            )}
        </div>
    )
}
