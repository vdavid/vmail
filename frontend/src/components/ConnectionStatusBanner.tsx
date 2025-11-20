import { useConnectionStore } from '../store/connection.store'

export default function ConnectionStatusBanner() {
    const { status, triggerReconnect } = useConnectionStore()

    if (status !== 'disconnected') {
        return null
    }

    return (
        <div className='bg-amber-100 px-3 py-2 text-sm text-amber-900'>
            <div className='mx-auto flex max-w-4xl items-center justify-between gap-3'>
                <span>Connection lost. New emails may be delayed.</span>
                <button
                    type='button'
                    className='underline'
                    onClick={() => {
                        triggerReconnect()
                    }}
                >
                    Try now
                </button>
            </div>
        </div>
    )
}
