import { Link } from 'react-router-dom'
import type { Thread } from '../lib/api'

interface EmailListItemProps {
    thread: Thread
    isSelected?: boolean
}

export default function EmailListItem({ thread, isSelected }: EmailListItemProps) {
    // Get the first message for display purposes
    const firstMessage = thread.messages?.[0]
    const sender = firstMessage?.from_address || 'Unknown'
    const subject = thread.subject || '(No subject)'
    const date = firstMessage?.sent_at ? new Date(firstMessage.sent_at).toLocaleDateString() : ''
    const threadCount = thread.messages?.length || 1
    const isUnread = firstMessage ? !firstMessage.is_read : false
    const isStarred = firstMessage?.is_starred || false

    return (
        <Link
            to={`/thread/${thread.stable_thread_id}`}
            className={`flex items-center gap-4 border-b border-gray-200 px-4 py-3 hover:bg-gray-50 ${
                isSelected ? 'bg-blue-50' : ''
            } ${isUnread ? 'font-semibold' : ''}`}
        >
            <div className='flex-shrink-0'>
                {isStarred ? (
                    <span className='text-yellow-500' aria-label='Starred'>
                        ★
                    </span>
                ) : (
                    <span className='text-gray-300' aria-label='Not starred'>
                        ☆
                    </span>
                )}
            </div>
            <div className='flex-1 min-w-0'>
                <div className='flex items-center gap-2'>
                    <span className='truncate text-sm text-gray-900'>{sender}</span>
                    {threadCount > 1 && (
                        <span className='text-xs text-gray-500'>({threadCount})</span>
                    )}
                </div>
                <div className='truncate text-sm text-gray-600'>{subject}</div>
            </div>
            <div className='flex-shrink-0 text-xs text-gray-500'>{date}</div>
        </Link>
    )
}
