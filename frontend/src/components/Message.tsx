/* eslint-disable react-dom/no-dangerously-set-innerhtml,react/no-danger --- We need dangerouslySetInnerHTML for HTML rendering.
   We sanitize the content so it should be safe. */
import DOMPurify from 'dompurify'

import { downloadAttachment, type Attachment, type Message as MessageType } from '../lib/api'

interface MessageProps {
    message: MessageType
}

export default function Message({ message }: MessageProps) {
    // Sanitize the HTML content before rendering
    const sanitizedHTML = message.unsafe_body_html ? DOMPurify.sanitize(message.unsafe_body_html) : ''

    const formatDate = (dateString: string | null) => {
        if (!dateString) return ''
        const date = new Date(dateString)
        return date.toLocaleString()
    }

    // Show all attachments - inline ones that aren't referenced in HTML should still be downloadable
    const attachments = message.attachments ?? []

    const handleDownload = async (attachment: Attachment) => {
        try {
            await downloadAttachment(message.id, attachment.id, attachment.filename)
        } catch (error) {
            // eslint-disable-next-line no-console
            console.error('Failed to download attachment:', error)
        }
    }

    return (
        <article className='rounded-3xl border border-white/5 bg-slate-950/50 p-5 text-slate-100 shadow-[0_25px_50px_-12px_rgba(15,23,42,0.8)]'>
            <header className='flex flex-col gap-3 border-b border-white/5 pb-4 md:flex-row md:items-start md:justify-between'>
                <div>
                    <p className='text-sm uppercase tracking-wide text-slate-400'>From</p>
                    <p className='text-lg font-semibold text-white'>{message.from_address}</p>
                    <p className='text-xs text-slate-400'>
                        To: <span className='text-slate-200'>{message.to_addresses.join(', ')}</span>
                    </p>
                    {message.cc_addresses.length > 0 && (
                        <p className='text-xs text-slate-400'>
                            CC: <span className='text-slate-200'>{message.cc_addresses.join(', ')}</span>
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
                <div className='mt-4 rounded-2xl border border-white/10 bg-white/5 p-4 text-sm'>
                    <p className='mb-3 font-semibold text-white'>Attachments</p>
                    <div className='flex flex-wrap gap-2'>
                        {attachments.map((attachment) => (
                            <button
                                key={attachment.id}
                                type='button'
                                onClick={() => void handleDownload(attachment)}
                                className='flex cursor-pointer items-center gap-2 rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-slate-200 transition-colors hover:border-white/20 hover:bg-white/10'
                            >
                                <span className='text-lg'>{getFileIcon(attachment.mime_type)}</span>
                                <span className='max-w-[150px] truncate'>{attachment.filename}</span>
                                <span className='text-xs text-slate-400'>{formatFileSize(attachment.size_bytes)}</span>
                            </button>
                        ))}
                    </div>
                </div>
            )}
            {sanitizedHTML ? (
                <div
                    className='prose prose-sm max-w-none text-slate-100'
                    dangerouslySetInnerHTML={{ __html: sanitizedHTML }}
                />
            ) : (
                message.body_text && <div className='mt-4 whitespace-pre-wrap text-slate-100'>{message.body_text}</div>
            )}
        </article>
    )
}

function formatFileSize(bytes: number): string {
    if (bytes < 1024) return `${String(bytes)} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function getFileIcon(mimeType: string): string {
    if (mimeType.startsWith('image/')) return '🖼️'
    if (mimeType.startsWith('video/')) return '🎬'
    if (mimeType.startsWith('audio/')) return '🎵'
    if (mimeType === 'application/pdf') return '📄'
    if (mimeType.includes('zip') || mimeType.includes('archive') || mimeType.includes('compressed')) return '📦'
    if (mimeType.includes('spreadsheet') || mimeType.includes('excel')) return '📊'
    if (mimeType.includes('document') || mimeType.includes('word')) return '📝'
    if (mimeType.includes('presentation') || mimeType.includes('powerpoint')) return '📽️'
    return '📎'
}
