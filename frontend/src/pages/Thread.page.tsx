import { useQuery } from '@tanstack/react-query'
import { useParams, useNavigate } from 'react-router-dom'

import Message from '../components/Message'
import { api } from '../lib/api'

export default function ThreadPage() {
    const { threadId } = useParams<{ threadId: string }>()
    const navigate = useNavigate()

    const {
        data: thread,
        isLoading,
        error,
    } = useQuery({
        queryKey: ['thread', threadId],
        queryFn: () => {
            if (!threadId) {
                throw new Error('Thread ID is required')
            }
            return api.getThread(threadId)
        },
        enabled: !!threadId,
    })

    const handleBack = () => {
        void navigate('/')
    }

    if (isLoading) {
        return (
            <div className='p-6'>
                <button
                    onClick={handleBack}
                    className='mb-4 text-sm text-blue-600 hover:text-blue-800'
                >
                    ← Back to Inbox
                </button>
                <p className='mt-4 text-gray-600'>Loading...</p>
            </div>
        )
    }

    if (error) {
        return (
            <div className='p-6'>
                <button
                    onClick={handleBack}
                    className='mb-4 text-sm text-blue-600 hover:text-blue-800'
                >
                    ← Back to Inbox
                </button>
                <p className='mt-4 text-red-600'>Error loading thread: {error.message}</p>
            </div>
        )
    }

    if (!thread) {
        return (
            <div className='p-6'>
                <button
                    onClick={handleBack}
                    className='mb-4 text-sm text-blue-600 hover:text-blue-800'
                >
                    ← Back to Inbox
                </button>
                <p className='mt-4 text-gray-600'>Thread not found</p>
            </div>
        )
    }

    return (
        <div className='flex h-full flex-col'>
            <div className='border-b border-gray-200 px-6 py-4'>
                <button
                    onClick={handleBack}
                    className='mb-2 text-sm text-blue-600 hover:text-blue-800'
                >
                    ← Back to Inbox
                </button>
                <h1 className='text-2xl font-bold text-gray-900'>
                    {thread.subject || '(No subject)'}
                </h1>
            </div>
            <div className='flex-1 overflow-y-auto'>
                {thread.messages && thread.messages.length > 0 ? (
                    thread.messages.map((message) => <Message key={message.id} message={message} />)
                ) : (
                    <div className='p-6 text-center text-gray-500'>No messages found</div>
                )}
            </div>
        </div>
    )
}
