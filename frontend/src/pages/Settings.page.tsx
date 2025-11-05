import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, type UserSettings } from '../lib/api'
import * as React from 'react'

const defaultSettings: UserSettings = {
    imap_server_hostname: '',
    imap_username: '',
    imap_password: '',
    smtp_server_hostname: '',
    smtp_username: '',
    smtp_password: '',
    archive_folder_name: 'Archive',
    sent_folder_name: 'Sent',
    drafts_folder_name: 'Drafts',
    trash_folder_name: 'Trash',
    spam_folder_name: 'Spam',
    undo_send_delay_seconds: 20,
    pagination_threads_per_page: 100,
}

export default function SettingsPage() {
    const queryClient = useQueryClient()
    const [formData, setFormData] = useState<UserSettings>(defaultSettings)
    const [saveMessage, setSaveMessage] = useState<string | null>(null)

    const { data, isLoading, isError } = useQuery<UserSettings>({
        queryKey: ['settings'],
        queryFn: api.getSettings,
        retry: false,
    })

    useEffect(() => {
        if (data) {
            setFormData({
                ...data,
                imap_password: '',
                smtp_password: '',
            })
        } else if (isError) {
            setFormData(defaultSettings)
        }
    }, [data, isError])

    const saveMutation = useMutation({
        mutationFn: api.saveSettings,
        onSuccess: () => {
            void queryClient.invalidateQueries({ queryKey: ['settings'] })
            void queryClient.invalidateQueries({ queryKey: ['authStatus'] })
            setSaveMessage('Settings saved successfully')
            setTimeout(() => setSaveMessage(null), 3_000)
        },
        onError: (error: Error) => {
            setSaveMessage(`Error: ${error.message}`)
            setTimeout(() => setSaveMessage(null), 5_000)
        },
    })

    const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
        const { name, value, type } = e.target
        setFormData((prev) => ({
            ...prev,
            [name]: type === 'number' ? parseInt(value) || 0 : value,
        }))
    }

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault()
        saveMutation.mutate(formData)
    }

    if (isLoading) {
        return (
            <div className='flex h-full items-center justify-center'>
                <div className='text-gray-600'>Loading settings...</div>
            </div>
        )
    }

    return (
        <div className='mx-auto max-w-3xl p-6'>
            <h1 className='mb-6 text-3xl font-bold text-gray-900'>Settings</h1>

            {saveMessage && (
                <div
                    className={`mb-4 rounded-md p-4 ${
                        saveMessage.startsWith('Error')
                            ? 'bg-red-50 text-red-800'
                            : 'bg-green-50 text-green-800'
                    }`}
                >
                    {saveMessage}
                </div>
            )}

            <form onSubmit={handleSubmit} className='space-y-8'>
                <section>
                    <h2 className='mb-4 text-xl font-semibold text-gray-900'>IMAP settings</h2>
                    <div className='space-y-4'>
                        <div>
                            <label
                                htmlFor='imap_server_hostname'
                                className='block text-sm font-medium text-gray-700'
                            >
                                IMAP server
                            </label>
                            <input
                                type='text'
                                id='imap_server_hostname'
                                name='imap_server_hostname'
                                value={formData.imap_server_hostname}
                                onChange={handleChange}
                                required
                                placeholder='Example: imap.example.com:993'
                                className='mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500'
                            />
                        </div>
                        <div>
                            <label
                                htmlFor='imap_username'
                                className='block text-sm font-medium text-gray-700'
                            >
                                IMAP username
                            </label>
                            <input
                                type='text'
                                id='imap_username'
                                name='imap_username'
                                value={formData.imap_username}
                                onChange={handleChange}
                                required
                                placeholder='Example: user@example.com'
                                className='mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500'
                            />
                        </div>
                        <div>
                            <label
                                htmlFor='imap_password'
                                className='block text-sm font-medium text-gray-700'
                            >
                                IMAP password
                            </label>
                            <input
                                type='password'
                                id='imap_password'
                                name='imap_password'
                                value={formData.imap_password}
                                onChange={handleChange}
                                required
                                placeholder='Enter password'
                                className='mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500'
                            />
                        </div>
                    </div>
                </section>

                <section>
                    <h2 className='mb-4 text-xl font-semibold text-gray-900'>SMTP settings</h2>
                    <div className='space-y-4'>
                        <div>
                            <label
                                htmlFor='smtp_server_hostname'
                                className='block text-sm font-medium text-gray-700'
                            >
                                SMTP server
                            </label>
                            <input
                                type='text'
                                id='smtp_server_hostname'
                                name='smtp_server_hostname'
                                value={formData.smtp_server_hostname}
                                onChange={handleChange}
                                required
                                placeholder='Example: smtp.example.com:587'
                                className='mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500'
                            />
                        </div>
                        <div>
                            <label
                                htmlFor='smtp_username'
                                className='block text-sm font-medium text-gray-700'
                            >
                                SMTP username
                            </label>
                            <input
                                type='text'
                                id='smtp_username'
                                name='smtp_username'
                                value={formData.smtp_username}
                                onChange={handleChange}
                                required
                                placeholder='Example: user@example.com'
                                className='mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500'
                            />
                        </div>
                        <div>
                            <label
                                htmlFor='smtp_password'
                                className='block text-sm font-medium text-gray-700'
                            >
                                SMTP password
                            </label>
                            <input
                                type='password'
                                id='smtp_password'
                                name='smtp_password'
                                value={formData.smtp_password}
                                onChange={handleChange}
                                required
                                placeholder='Enter password'
                                className='mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500'
                            />
                        </div>
                    </div>
                </section>

                <section>
                    <h2 className='mb-4 text-xl font-semibold text-gray-900'>Folder names</h2>
                    <div className='grid grid-cols-2 gap-4'>
                        <div>
                            <label
                                htmlFor='archive_folder_name'
                                className='block text-sm font-medium text-gray-700'
                            >
                                Archive folder
                            </label>
                            <input
                                type='text'
                                id='archive_folder_name'
                                name='archive_folder_name'
                                value={formData.archive_folder_name}
                                onChange={handleChange}
                                required
                                className='mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500'
                            />
                        </div>
                        <div>
                            <label
                                htmlFor='sent_folder_name'
                                className='block text-sm font-medium text-gray-700'
                            >
                                Sent folder
                            </label>
                            <input
                                type='text'
                                id='sent_folder_name'
                                name='sent_folder_name'
                                value={formData.sent_folder_name}
                                onChange={handleChange}
                                required
                                className='mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500'
                            />
                        </div>
                        <div>
                            <label
                                htmlFor='drafts_folder_name'
                                className='block text-sm font-medium text-gray-700'
                            >
                                Drafts folder
                            </label>
                            <input
                                type='text'
                                id='drafts_folder_name'
                                name='drafts_folder_name'
                                value={formData.drafts_folder_name}
                                onChange={handleChange}
                                required
                                className='mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500'
                            />
                        </div>
                        <div>
                            <label
                                htmlFor='trash_folder_name'
                                className='block text-sm font-medium text-gray-700'
                            >
                                Trash folder
                            </label>
                            <input
                                type='text'
                                id='trash_folder_name'
                                name='trash_folder_name'
                                value={formData.trash_folder_name}
                                onChange={handleChange}
                                required
                                className='mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500'
                            />
                        </div>
                        <div>
                            <label
                                htmlFor='spam_folder_name'
                                className='block text-sm font-medium text-gray-700'
                            >
                                Spam folder
                            </label>
                            <input
                                type='text'
                                id='spam_folder_name'
                                name='spam_folder_name'
                                value={formData.spam_folder_name}
                                onChange={handleChange}
                                required
                                className='mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500'
                            />
                        </div>
                    </div>
                </section>

                <section>
                    <h2 className='mb-4 text-xl font-semibold text-gray-900'>Preferences</h2>
                    <div className='grid grid-cols-2 gap-4'>
                        <div>
                            <label
                                htmlFor='undo_send_delay_seconds'
                                className='block text-sm font-medium text-gray-700'
                            >
                                Undo send delay (seconds)
                            </label>
                            <input
                                type='number'
                                id='undo_send_delay_seconds'
                                name='undo_send_delay_seconds'
                                value={formData.undo_send_delay_seconds}
                                onChange={handleChange}
                                min='0'
                                max='60'
                                required
                                className='mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500'
                            />
                        </div>
                        <div>
                            <label
                                htmlFor='pagination_threads_per_page'
                                className='block text-sm font-medium text-gray-700'
                            >
                                Threads per page
                            </label>
                            <input
                                type='number'
                                id='pagination_threads_per_page'
                                name='pagination_threads_per_page'
                                value={formData.pagination_threads_per_page}
                                onChange={handleChange}
                                min='10'
                                max='200'
                                required
                                className='mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500'
                            />
                        </div>
                    </div>
                </section>

                <div className='flex justify-end'>
                    <button
                        type='submit'
                        disabled={saveMutation.isPending}
                        className='rounded-md bg-blue-600 px-6 py-2 text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50'
                    >
                        {saveMutation.isPending ? 'Saving...' : 'Save Settings'}
                    </button>
                </div>
            </form>
        </div>
    )
}
