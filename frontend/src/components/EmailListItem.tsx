import * as React from 'react'
import { Link } from 'react-router-dom'

import type { Thread } from '../lib/api'
import { encodeThreadIdForUrl } from '../lib/api'

interface EmailListItemProps {
    thread: Thread
    isSelected?: boolean
}

const timeFormatter = new Intl.DateTimeFormat(undefined, {
    hour: 'numeric',
    minute: '2-digit',
})

const dateFormatter = new Intl.DateTimeFormat(undefined, {
    month: 'short',
    day: 'numeric',
})

function formatDate(sentAt: string | null | undefined): string {
    if (!sentAt) {
        return ''
    }
    const date = new Date(sentAt)
    const now = new Date()
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
    const messageDate = new Date(date.getFullYear(), date.getMonth(), date.getDate())

    if (messageDate.getTime() === today.getTime()) {
        return timeFormatter.format(date)
    }
    return dateFormatter.format(date)
}

function getPreviewSnippet(
    previewSnippet: string | undefined,
    bodyText: string | undefined,
): string {
    // Prefer backend preview_snippet, fallback to first message body_text
    const source = previewSnippet || bodyText
    if (!source) {
        return ''
    }
    const snippetText = source.replace(/\s+/g, ' ').trim()
    if (snippetText.length === 0) {
        return ''
    }
    // Backend already limits to 100 chars, but we'll truncate further if needed for display
    return snippetText.length > 100 ? snippetText.slice(0, 100) + '...' : snippetText
}

function extractSenderName(fromAddress: string): string {
    // Extract name from "Name <email@domain.com>" format
    // If no angle brackets, assume it's just an email address
    const match = fromAddress.match(/^(.+?)\s*<.+>$/)
    if (match) {
        return match[1].trim()
    }
    // If no name found, return the original (might be just email)
    return fromAddress
}

function truncateSenderName(name: string): string {
    // If exactly 20 characters, display it fully
    if (name.length === 20) {
        return name
    }
    // If longer than 19, truncate to 19 and add "."
    if (name.length > 19) {
        return name.slice(0, 19) + '.'
    }
    // Otherwise, return as-is
    return name
}

function getLinkClassName(isSelected: boolean, isUnread: boolean): string {
    const baseClasses =
        'grid grid-cols-[24px_32px_200px_1fr_24px_80px] items-center gap-2 px-4 py-2 sm:px-6 hover:bg-white/5 focus-visible:outline-2 focus-visible:outline-blue-400'
    if (isSelected) {
        return `${baseClasses} bg-blue-500/20 text-white`
    }
    if (isUnread) {
        return `${baseClasses} bg-white/0 text-white`
    }
    return `${baseClasses} text-slate-300`
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

// eslint-disable-next-line complexity -- It's okay to have a lot of logic here
export default function EmailListItem({ thread, isSelected }: EmailListItemProps) {
    const firstMessage = thread.messages?.[0]
    const fromAddress = thread.first_message_from_address || firstMessage?.from_address || 'Unknown'
    const senderName = truncateSenderName(extractSenderName(fromAddress))
    const subject = thread.subject || '(No subject)'
    const threadCount = thread.message_count || 1
    const isUnread = firstMessage ? !firstMessage.is_read : false
    const isStarred = firstMessage?.is_starred || false
    const previewSnippet = getPreviewSnippet(thread.preview_snippet, firstMessage?.body_text)
    // Use last_sent_at from thread (for list views) or fallback to first message sent_at (for detail views)
    const dateToFormat = thread.last_sent_at || firstMessage?.sent_at || null
    const formattedDate = formatDate(dateToFormat)
    const hasAttachments = thread.has_attachments

    const senderClassName = `truncate text-sm ${isUnread ? 'font-semibold text-white' : 'text-slate-100'}`
    const subjectClassName = `truncate text-sm ${isUnread ? 'font-semibold text-white' : 'text-slate-300'}`

    return (
        <Link
            to={`/thread/${encodeThreadIdForUrl(thread.stable_thread_id)}`}
            className={getLinkClassName(isSelected ?? false, isUnread)}
            onKeyDown={(e) => {
                handleKeyDown(e, isSelected ?? false)
            }}
            tabIndex={isSelected ? -1 : 0}
        >
            {/* Checkbox column */}
            <div className='flex items-center justify-center'>
                <input
                    type='checkbox'
                    className='h-4 w-4 cursor-pointer rounded border-white/20 bg-slate-900/50 text-blue-500 focus:ring-2 focus:ring-blue-400'
                    onClick={(e) => {
                        e.preventDefault()
                        e.stopPropagation()
                    }}
                    aria-label='Select email'
                />
            </div>

            {/* Star column */}
            <div className='flex justify-center text-base'>
                <span
                    className={isStarred ? 'text-amber-400' : 'text-slate-500'}
                    aria-hidden='true'
                >
                    {isStarred ? '★' : '☆'}
                </span>
            </div>

            {/* Sender column with thread count */}
            <div className='min-w-0'>
                <span data-testid='email-sender' className={senderClassName}>
                    {senderName}
                    {threadCount > 1 && (
                        <>
                            {' '}
                            <small className='text-xs opacity-60'>{threadCount}</small>
                        </>
                    )}
                </span>
            </div>

            {/* Subject and preview column */}
            <div className='min-w-0'>
                <div className='flex items-center gap-2 text-sm'>
                    <span data-testid='email-subject' className={subjectClassName}>
                        {subject}
                    </span>
                    {previewSnippet && (
                        <>
                            <span className='opacity-60 text-slate-400'> - </span>
                            <span className='truncate opacity-60 text-slate-400'>
                                {previewSnippet}
                            </span>
                        </>
                    )}
                </div>
            </div>

            {/* Attachment indicator column */}
            <div className='flex justify-center text-slate-400'>
                {hasAttachments && <span aria-label='Has attachments'>📎</span>}
            </div>

            {/* Date column */}
            <div className='text-right text-xs text-slate-400'>{formattedDate}</div>
        </Link>
    )
}
