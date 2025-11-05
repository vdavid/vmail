import { Link, useLocation } from 'react-router-dom'

const navigation = [
    { name: 'Inbox', href: '/?folder=INBOX', icon: '📥' },
    { name: 'Starred', href: '/?folder=Starred', icon: '⭐' },
    { name: 'Sent', href: '/?folder=Sent', icon: '📤' },
    { name: 'Drafts', href: '/?folder=Drafts', icon: '📝' },
    { name: 'Spam', href: '/?folder=Spam', icon: '🚫' },
    { name: 'Trash', href: '/?folder=Trash', icon: '🗑️' },
]

export default function Sidebar() {
    const location = useLocation()

    return (
        <div className='flex h-full w-64 flex-col border-r border-gray-200 bg-white'>
            <div className='flex h-16 items-center border-b border-gray-200 px-6'>
                <h1 className='text-xl font-bold text-gray-900'>V-Mail</h1>
            </div>
            <nav className='flex-1 space-y-1 px-3 py-4' aria-label='Sidebar'>
                {navigation.map((item) => {
                    const isActive =
                        location.pathname === '/' && location.search === item.href.split('?')[1]
                    return (
                        <Link
                            key={item.name}
                            to={item.href}
                            className={`group flex items-center rounded-md px-3 py-2 text-sm font-medium ${
                                isActive
                                    ? 'bg-gray-100 text-gray-900'
                                    : 'text-gray-700 hover:bg-gray-50 hover:text-gray-900'
                            }`}
                            aria-current={isActive ? 'page' : undefined}
                        >
                            <span className='mr-3 text-lg' aria-hidden='true'>
                                {item.icon}
                            </span>
                            {item.name}
                        </Link>
                    )
                })}
            </nav>
            <div className='border-t border-gray-200 p-4'>
                <Link
                    to='/settings'
                    className='flex items-center text-sm text-gray-700 hover:text-gray-900'
                >
                    <span className='mr-3 text-lg' aria-hidden='true'>
                        ⚙️
                    </span>
                    Settings
                </Link>
            </div>
        </div>
    )
}
