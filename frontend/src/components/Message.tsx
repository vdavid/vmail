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

    return (
        <div className='border-b border-gray-200 p-6'>
            <div className='mb-4'>
                <div className='flex items-center justify-between'>
                    <div>
                        <div className='font-semibold text-gray-900'>{message.from_address}</div>
                        <div className='text-sm text-gray-600'>
                            To: {message.to_addresses.join(', ')}
                        </div>
                        {message.cc_addresses.length > 0 && (
                            <div className='text-sm text-gray-600'>
                                CC: {message.cc_addresses.join(', ')}
                            </div>
                        )}
                    </div>
                    <div className='text-sm text-gray-500'>{formatDate(message.sent_at)}</div>
                </div>
                {message.subject && (
                    <div className='mt-2 font-semibold text-gray-900'>{message.subject}</div>
                )}
            </div>
            {message.attachments && message.attachments.length > 0 && (
                <div className='mb-4'>
                    <div className='text-sm font-semibold text-gray-700'>Attachments:</div>
                    <ul className='mt-1 space-y-1'>
                        {message.attachments
                            .filter((att) => !att.is_inline)
                            .map((attachment) => (
                                <li key={attachment.id} className='text-sm text-blue-600'>
                                    {attachment.filename} ({formatFileSize(attachment.size_bytes)})
                                </li>
                            ))}
                    </ul>
                </div>
            )}
            <div
                className='prose prose-sm max-w-none text-gray-900'
                dangerouslySetInnerHTML={{ __html: sanitizedHTML }}
            />
            {!sanitizedHTML && message.body_text && (
                <div className='whitespace-pre-wrap text-gray-900'>{message.body_text}</div>
            )}
        </div>
    )
}

function formatFileSize(bytes: number): string {
    if (bytes < 1024) return `${String(bytes)} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}
