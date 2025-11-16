/* eslint-disable react-dom/no-dangerously-set-innerhtml,react/no-danger --- We need dangerouslySetInnerHTML for HTML rendering.
   We sanitize the content so it should be safe. */
import DOMPurify from 'dompurify'

import type { Message as MessageType } from '../lib/api'

interface MessageProps {
    message: MessageType
}

export default function Message({ message }: MessageProps) {
    // Sanitize the HTML content before rendering
    const sanitizedHTML = message.unsafe_body_html
        ? DOMPurify.sanitize(message.unsafe_body_html)
        : ''

    const formatDate = (dateString: string | null) => {
        if (!dateString) return ''
        const date = new Date(dateString)
        return date.toLocaleString()
    }

    const attachments = message.attachments?.filter((att) => !att.is_inline) ?? []

    return (
        <article className='rounded-3xl border border-white/5 bg-slate-950/50 p-5 text-slate-100 shadow-[0_25px_50px_-12px_rgba(15,23,42,0.8)]'>
            <header className='flex flex-col gap-3 border-b border-white/5 pb-4 md:flex-row md:items-start md:justify-between'>
                <div>
                    <p className='text-sm uppercase tracking-wide text-slate-400'>From</p>
                    <p className='text-lg font-semibold text-white'>{message.from_address}</p>
                    <p className='text-xs text-slate-400'>
                        To:{' '}
                        <span className='text-slate-200'>{message.to_addresses.join(', ')}</span>
                    </p>
                    {message.cc_addresses.length > 0 && (
                        <p className='text-xs text-slate-400'>
                            CC:{' '}
                            <span className='text-slate-200'>
                                {message.cc_addresses.join(', ')}
                            </span>
                        </p>
                    )}
                </div>
                <div className='text-sm text-slate-400'>{formatDate(message.sent_at)}</div>
            </header>
            {message.subject && (
                <div className='mt-4 rounded-2xl bg-white/5 px-4 py-2 text-sm font-semibold text-white'>
                    {message.subject}
                </div>
            )}
            {attachments.length > 0 && (
                <div className='mt-4 rounded-2xl border border-white/10 bg-white/5 p-3 text-sm'>
                    <p className='font-semibold text-white'>Attachments</p>
                    <ul className='mt-2 space-y-1'>
                        {attachments.map((attachment) => (
                            <li
                                key={attachment.id}
                                className='flex items-center justify-between text-slate-200'
                            >
                                <span>{attachment.filename}</span>
                                <span className='text-xs text-slate-400'>
                                    {formatFileSize(attachment.size_bytes)}
                                </span>
                            </li>
                        ))}
                    </ul>
                </div>
            )}
            {sanitizedHTML ? (
                <div
                    className='prose prose-sm max-w-none text-slate-100'
                    dangerouslySetInnerHTML={{ __html: sanitizedHTML }}
                />
            ) : (
                message.body_text && (
                    <div className='mt-4 whitespace-pre-wrap text-slate-100'>
                        {message.body_text}
                    </div>
                )
            )}
        </article>
    )
}

function formatFileSize(bytes: number): string {
    if (bytes < 1024) return `${String(bytes)} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}
