import { useSearchParams } from 'react-router-dom'

export default function InboxPage() {
    const [searchParams] = useSearchParams()
    const folder = searchParams.get('folder') || 'INBOX'

    return (
        <div className='p-6'>
            <h1 className='text-2xl font-bold text-gray-900'>
                {folder === 'INBOX' ? 'Inbox' : folder}
            </h1>
            <p className='mt-4 text-gray-600'>Thread list will be displayed here.</p>
        </div>
    )
}
