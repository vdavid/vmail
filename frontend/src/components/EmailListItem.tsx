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

function getSnippet(bodyText: string | undefined): string {
    if (!bodyText) {
        return ''
    }
    const snippetText = bodyText.replace(/\s+/g, ' ').trim()
    if (snippetText.length === 0) {
        return ''
    }
    return snippetText.length > 80 ? snippetText.slice(0, 80) + '...' : snippetText
}

function getFormattedDate(sentAt: string | undefined): string {
    if (!sentAt) {
        return ''
    }
    return dateFormatter.format(new Date(sentAt))
}

function getLinkClassName(isSelected: boolean, isUnread: boolean): string {
    const baseClasses =
        'flex items-center gap-3 px-4 py-2 sm:px-6 hover:bg-white/5 focus-visible:outline-2 focus-visible:outline-blue-400'
    if (isSelected) {
        return `${baseClasses} bg-blue-500/20 text-white`
    }
    if (isUnread) {
        return `${baseClasses} bg-white/0 text-white`
    }
    return `${baseClasses} text-slate-300`
}

function getTextClassName(isUnread: boolean, variant: 'sender' | 'subject'): string {
    const baseClass = 'truncate'
    if (isUnread) {
        return `${baseClass} font-semibold text-white`
    }
    return variant === 'sender' ? `${baseClass} text-slate-100` : `${baseClass} text-slate-300`
}

function StarIcon({ isStarred }: { isStarred: boolean }) {
    return (
        <div className='flex w-8 justify-center text-base'>
            <span className={isStarred ? 'text-amber-400' : 'text-slate-500'} aria-hidden='true'>
                {isStarred ? '★' : '☆'}
            </span>
        </div>
    )
}

function EmailContent({
    sender,
    subject,
    threadCount,
    snippet,
    isUnread,
}: {
    sender: string
    subject: string
    threadCount: number
    snippet: string
    isUnread: boolean
}) {
    return (
        <div className='min-w-0 flex-1'>
            <div className='flex items-center gap-2 text-sm'>
                <span data-testid='email-sender' className={getTextClassName(isUnread, 'sender')}>
                    {sender}
                </span>
                {threadCount > 1 && (
                    <span className='rounded-full border border-white/10 px-2 py-0.5 text-xs text-slate-300'>
                        {threadCount}
                    </span>
                )}
                <span className='text-slate-400'>—</span>
                <span data-testid='email-subject' className={getTextClassName(isUnread, 'subject')}>
                    {subject}
                </span>
                {snippet && <span className='truncate text-slate-400'> — {snippet}</span>}
            </div>
        </div>
    )
}

function isNavigationKey(key: string): boolean {
    return key === 'j' || key === 'k' || key === 'ArrowDown' || key === 'ArrowUp'
}

function handleKeyDown(event: React.KeyboardEvent<HTMLAnchorElement>, isSelected: boolean): void {
    if (isNavigationKey(event.key)) {
        event.preventDefault()
        event.stopPropagation()
        return
    }
    if (event.key === 'Enter' && isSelected) {
        event.preventDefault()
        event.stopPropagation()
    }
}

// eslint-disable-next-line complexity -- It's okay.
export default function EmailListItem({ thread, isSelected }: EmailListItemProps) {
    const firstMessage = thread.messages?.[0]
    const sender = thread.first_message_from_address || firstMessage?.from_address || 'Unknown'
    const subject = thread.subject || '(No subject)'
    const threadCount = thread.messages?.length || 1
    const isUnread = firstMessage ? !firstMessage.is_read : false
    const isStarred = firstMessage?.is_starred || false
    const snippet = getSnippet(firstMessage?.body_text)
    const formattedDate = getFormattedDate(firstMessage?.sent_at || undefined)

    return (
        <Link
            to={`/thread/${encodeThreadIdForUrl(thread.stable_thread_id)}`}
            className={getLinkClassName(isSelected ?? false, isUnread)}
            onKeyDown={(e) => {
                handleKeyDown(e, isSelected ?? false)
            }}
            tabIndex={isSelected ? -1 : 0}
        >
            <StarIcon isStarred={isStarred} />
            <EmailContent
                sender={sender}
                subject={subject}
                threadCount={threadCount}
                snippet={snippet}
                isUnread={isUnread}
            />
            <div className='w-16 flex-shrink-0 text-right text-xs text-slate-400'>
                {formattedDate}
            </div>
        </Link>
    )
}
