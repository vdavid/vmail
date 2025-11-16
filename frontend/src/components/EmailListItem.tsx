import * as React from 'react'
import { Link } from 'react-router-dom'

import type { Thread } from '../lib/api'
import { encodeThreadIdForUrl } from '../lib/api'

interface EmailListItemProps {
    thread: Thread
    isSelected?: boolean
}

const dateFormatter = new Intl.DateTimeFormat(undefined, {
    month: 'short',
    day: 'numeric',
})

export default function EmailListItem({ thread, isSelected }: EmailListItemProps) {
    const firstMessage = thread.messages?.[0]
    const sender = thread.first_message_from_address || firstMessage?.from_address || 'Unknown'
    const subject = thread.subject || '(No subject)'
    const threadCount = thread.messages?.length || 1
    const isUnread = firstMessage ? !firstMessage.is_read : false
    const isStarred = firstMessage?.is_starred || false
    const snippetSource = firstMessage?.body_text || ''
    const snippetText = snippetSource.replace(/\s+/g, ' ').trim()
    const snippet =
        snippetText.length > 0
            ? snippetText.slice(0, 80) + (snippetText.length > 80 ? '...' : '')
            : ''
    const formattedDate = firstMessage?.sent_at
        ? dateFormatter.format(new Date(firstMessage.sent_at))
        : ''

    const handleKeyDown = (event: React.KeyboardEvent<HTMLAnchorElement>) => {
        if (
            event.key === 'j' ||
            event.key === 'k' ||
            event.key === 'ArrowDown' ||
            event.key === 'ArrowUp'
        ) {
            event.preventDefault()
            event.stopPropagation()
        }
        if (event.key === 'Enter' && isSelected) {
            event.preventDefault()
            event.stopPropagation()
        }
    }

    return (
        <Link
            to={`/thread/${encodeThreadIdForUrl(thread.stable_thread_id)}`}
            className={`flex items-center gap-3 px-4 py-2 sm:px-6 ${
                isSelected
                    ? 'bg-blue-500/20 text-white'
                    : isUnread
                      ? 'bg-white/0 text-white'
                      : 'text-slate-300'
            } hover:bg-white/5 focus-visible:outline-2 focus-visible:outline-blue-400`}
            onKeyDown={handleKeyDown}
            tabIndex={isSelected ? -1 : 0}
        >
            <div className='flex w-8 justify-center text-base'>
                <span
                    className={isStarred ? 'text-amber-400' : 'text-slate-500'}
                    aria-hidden='true'
                >
                    {isStarred ? '★' : '☆'}
                </span>
            </div>
            <div className='min-w-0 flex-1'>
                <div className='flex items-center gap-2 text-sm'>
                    <span
                        data-testid='email-sender'
                        className={`truncate ${isUnread ? 'font-semibold text-white' : 'text-slate-100'}`}
                    >
                        {sender}
                    </span>
                    {threadCount > 1 && (
                        <span className='rounded-full border border-white/10 px-2 py-0.5 text-xs text-slate-300'>
                            {threadCount}
                        </span>
                    )}
                    <span className='text-slate-400'>—</span>
                    <span
                        data-testid='email-subject'
                        className={`truncate ${isUnread ? 'font-semibold text-white' : 'text-slate-300'}`}
                    >
                        {subject}
                    </span>
                    {snippet && <span className='truncate text-slate-400'> — {snippet}</span>}
                </div>
            </div>
            <div className='w-16 flex-shrink-0 text-right text-xs text-slate-400'>
                {formattedDate}
            </div>
        </Link>
    )
}
