import { useQuery } from '@tanstack/react-query'
import { useParams, useNavigate } from 'react-router-dom'

import Message from '../components/Message'
import { api, decodeThreadIdFromUrl } from '../lib/api'

const LoadingState = (
    <div className='flex h-full flex-col gap-4 p-6 text-slate-200'>
        <div className='h-8 w-32 rounded-full bg-white/10' />
        <div className='rounded-3xl border border-white/5 bg-white/5 p-6 text-sm'>Loading...</div>
    </div>
)

interface ErrorStateProps {
    message: string
    onBack: () => void
}

function ErrorState({ message, onBack }: ErrorStateProps) {
    return (
        <div className='p-6 text-slate-100'>
            <button
                onClick={onBack}
                className='mb-4 inline-flex items-center gap-2 text-sm text-slate-300 transition hover:text-white'
            >
                <span aria-hidden='true'>←</span> Back to Inbox
            </button>
            <p className='mt-2 rounded-2xl bg-red-900/40 p-4 text-sm text-red-100'>{message}</p>
        </div>
    )
}

export default function ThreadPage() {
    const { threadId: encodedThreadId } = useParams<{ threadId: string }>()
    const navigate = useNavigate()

    // Decode the base64 URL-safe thread ID to get the raw Message-ID
    let rawThreadId: string | null = null
    let decodeError: Error | null = null

    if (encodedThreadId) {
        try {
            rawThreadId = decodeThreadIdFromUrl(encodedThreadId)
        } catch (e) {
            decodeError = e instanceof Error ? e : new Error(String(e))
        }
    }

    const {
        data: thread,
        isLoading,
        error,
    } = useQuery({
        queryKey: ['thread', rawThreadId],
        queryFn: () => {
            if (!rawThreadId) {
                throw new Error('Thread ID is required')
            }
            return api.getThread(rawThreadId)
        },
        enabled: !!rawThreadId && !decodeError,
    })

    const handleBack = () => {
        void navigate('/')
    }

    if (isLoading) {
        return LoadingState
    }

    if (decodeError) {
        return (
            <ErrorState
                message={`Error decoding thread ID: ${decodeError.message}`}
                onBack={handleBack}
            />
        )
    }

    if (error) {
        return <ErrorState message={`Error loading thread: ${error.message}`} onBack={handleBack} />
    }

    if (!thread) {
        return <ErrorState message='Thread not found' onBack={handleBack} />
    }

    const subject = thread.subject || '(No subject)'

    return (
        <div className='flex h-full flex-col text-white'>
            <div className='border-b border-white/5 px-4 py-4 sm:px-6'>
                <button
                    onClick={handleBack}
                    className='mb-2 inline-flex items-center gap-2 text-sm text-slate-300 transition hover:text-white'
                >
                    <span aria-hidden='true'>←</span> Back to Inbox
                </button>
                <h1 className='text-xl font-semibold text-white'>{subject}</h1>
                <p className='mt-1 text-sm text-slate-400'>Conversation view</p>
            </div>
            <div className='flex-1 overflow-y-auto px-4 py-4 sm:px-6'>
                {thread.messages && thread.messages.length > 0 ? (
                    <div className='flex flex-col gap-4'>
                        {thread.messages.map((message) => (
                            <Message key={message.id} message={message} />
                        ))}
                    </div>
                ) : (
                    <div className='rounded-3xl border border-white/5 bg-white/5 p-6 text-center text-sm text-slate-400'>
                        No messages found
                    </div>
                )}
            </div>
        </div>
    )
}
