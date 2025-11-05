import { useParams, useNavigate } from 'react-router-dom'

export default function ThreadPage() {
    const { threadId } = useParams<{ threadId: string }>()
    const navigate = useNavigate()

    const handleBack = () => {
        navigate('/')
    }

    return (
        <div className='p-6'>
            <button onClick={handleBack} className='mb-4 text-sm text-blue-600 hover:text-blue-800'>
                ← Back to Inbox
            </button>
            <h1 className='text-2xl font-bold text-gray-900'>Thread: {threadId}</h1>
            <p className='mt-4 text-gray-600'>Thread messages will be displayed here.</p>
        </div>
    )
}
