import { useQuery } from '@tanstack/react-query'
import { useNavigate, useSearchParams } from 'react-router-dom'

import { api } from '../lib/api'
import type { Pagination } from '../lib/api'

interface EmailListPaginationProps {
    pagination: Pagination
}

export default function EmailListPagination({ pagination }: EmailListPaginationProps) {
    const navigate = useNavigate()
    const [searchParams] = useSearchParams()
    const totalPages = Math.ceil(pagination.total_count / pagination.per_page)

    // Get folders to determine which folder is the inbox
    const { data: folders } = useQuery({
        queryKey: ['folders'],
        queryFn: () => api.getFolders(),
    })

    const handlePageChange = (newPage: number) => {
        const params = new URLSearchParams(searchParams)
        params.set('page', newPage.toString())

        // Remove folder param if it's the inbox folder (inbox uses just '/' as the URL)
        const folderParam = params.get('folder')
        if (folderParam) {
            const inboxFolder = folders?.find((f) => f.role === 'inbox')
            if (inboxFolder && folderParam === inboxFolder.name) {
                params.delete('folder')
            }
        }

        // Determine the base path based on whether we're on search or inbox
        const basePath = params.has('q') ? '/search' : '/'

        // Build the URL - if there are no params left (or only page param for INBOX), use basePath
        // Otherwise, append the query string
        const queryString = params.toString()
        const url = queryString ? `${basePath}?${queryString}` : basePath
        void navigate(url)
    }

    if (totalPages <= 1) {
        return null
    }

    return (
        <div className='flex items-center justify-between border-t border-gray-200 bg-white px-6 py-4'>
            <div className='text-sm text-gray-700'>
                Page {pagination.page} of {totalPages}
            </div>
            <div className='flex gap-2'>
                <button
                    onClick={() => {
                        handlePageChange(pagination.page - 1)
                    }}
                    disabled={pagination.page <= 1}
                    className='rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50'
                >
                    Previous
                </button>
                <button
                    onClick={() => {
                        handlePageChange(pagination.page + 1)
                    }}
                    disabled={pagination.page >= totalPages}
                    className='rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50'
                >
                    Next
                </button>
            </div>
        </div>
    )
}
