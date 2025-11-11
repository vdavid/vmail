import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as React from 'react'
import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'

import { api, type UserSettings } from '../lib/api'

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
    const navigate = useNavigate()
    const [saveMessage, setSaveMessage] = useState<string | null>(null)
    const initializedRef = useRef(false)
    const wasNewUserRef = useRef(false)

    const { data, isLoading, isError, error } = useQuery<UserSettings>({
        queryKey: ['settings'],
        queryFn: () => api.getSettings(),
        retry: false,
    })

    const [formData, setFormData] = useState<UserSettings>(defaultSettings)

    // Store raw string values for number inputs to allow for an empty state.
    // This hacky solution seems to be needed because otherwise, when momentarily clearing the input
    // to set it to a new value, the input auto-inserts 0, which is awkward.
    const [numberInputs, setNumberInputs] = useState<{
        undo_send_delay_seconds: string
        pagination_threads_per_page: string
    }>({
        undo_send_delay_seconds: '20',
        pagination_threads_per_page: '100',
    })

    // Initialize form data from query data when it first loads
    // Using a ref to ensure we only initialize once to avoid cascading renders
    useEffect(() => {
        if (data && !initializedRef.current) {
            initializedRef.current = true
            // eslint-disable-next-line react-hooks/set-state-in-effect -- Synchronizing external query state with form state
            setFormData({
                ...data,
                imap_password: '',
                smtp_password: '',
            })
            setNumberInputs({
                undo_send_delay_seconds: String(data.undo_send_delay_seconds),
                pagination_threads_per_page: String(data.pagination_threads_per_page),
            })
        } else if (isError && !initializedRef.current) {
            initializedRef.current = true

            // Only treat 404 (Not Found) as "new user" - other errors are real problems
            // that should not trigger redirect after save
            const fetchError = error as Error & { status?: number }

            wasNewUserRef.current = fetchError.status === 404 // Only true for 404 (settings not found)

            setFormData(defaultSettings)
            setNumberInputs({
                undo_send_delay_seconds: String(defaultSettings.undo_send_delay_seconds),
                pagination_threads_per_page: String(defaultSettings.pagination_threads_per_page),
            })
        }
    }, [data, isError, error])

    const saveMutation = useMutation({
        mutationFn: (settings: UserSettings) => api.saveSettings(settings),
        onSuccess: () => {
            void queryClient.invalidateQueries({ queryKey: ['settings'] })
            void queryClient.invalidateQueries({ queryKey: ['authStatus'] })
            // Invalidate threads queries so they refetch with new pagination limit
            void queryClient.invalidateQueries({ queryKey: ['threads'] })
            setSaveMessage('Settings saved successfully')

            // If this was a new user completing onboarding, redirect to inbox
            if (wasNewUserRef.current) {
                // Wait a bit for authStatus to update, then redirect
                setTimeout(() => {
                    void navigate('/', { replace: true })
                }, 500)
            } else {
                setTimeout(() => {
                    setSaveMessage(null)
                }, 3_000)
            }
        },
        onError: (error: Error) => {
            setSaveMessage(`Error: ${error.message}`)
            setTimeout(() => {
                setSaveMessage(null)
            }, 5_000)
        },
    })

    const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
        const { name, value, type } = e.target
        if (type === 'number') {
            // Store raw string value for number inputs
            setNumberInputs((prev) => ({
                ...prev,
                [name]: value,
            }))
            // Only update formData if the value is a valid number
            const numValue = value === '' ? 0 : parseInt(value, 10)
            if (!isNaN(numValue)) {
                setFormData((prev) => ({
                    ...prev,
                    [name]: numValue,
                }))
            }
        } else {
            setFormData((prev) => ({
                ...prev,
                [name]: value,
            }))
        }
    }

    const handleNumberBlur = (e: React.FocusEvent<HTMLInputElement>) => {
        const { name, value } = e.target
        const numValue = value === '' ? 0 : parseInt(value, 10)
        const finalValue = isNaN(numValue) ? 0 : numValue
        setNumberInputs((prev) => ({
            ...prev,
            [name]: String(finalValue),
        }))
        setFormData((prev) => ({
            ...prev,
            [name]: finalValue,
        }))
    }

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault()
        // Ensure number fields are integers before submitting
        const submitData = {
            ...formData,
            undo_send_delay_seconds: parseInt(numberInputs.undo_send_delay_seconds, 10) || 0,
            pagination_threads_per_page:
                parseInt(numberInputs.pagination_threads_per_page, 10) || 0,
        }
        saveMutation.mutate(submitData)
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
                                placeholder={
                                    formData.imap_password_set
                                        ? 'Password is set (leave empty to keep current)'
                                        : 'Enter password'
                                }
                                className='mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500'
                            />
                            {formData.imap_password_set && (
                                <p className='mt-1 text-sm text-gray-500'>
                                    Password is currently set. Leave empty to keep the existing
                                    password.
                                </p>
                            )}
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
                                placeholder={
                                    formData.smtp_password_set
                                        ? 'Password is set (leave empty to keep current)'
                                        : 'Enter password'
                                }
                                className='mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500'
                            />
                            {formData.smtp_password_set && (
                                <p className='mt-1 text-sm text-gray-500'>
                                    Password is currently set. Leave empty to keep the existing
                                    password.
                                </p>
                            )}
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
                                value={numberInputs.undo_send_delay_seconds}
                                onChange={handleChange}
                                onBlur={handleNumberBlur}
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
                                value={numberInputs.pagination_threads_per_page}
                                onChange={handleChange}
                                onBlur={handleNumberBlur}
                                min='5'
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
