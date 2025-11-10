import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Routes, Route } from 'react-router-dom'

import AuthWrapper from './components/AuthWrapper'
import Layout from './components/Layout'
import InboxPage from './pages/Inbox.page'
import SearchPage from './pages/Search.page'
import SettingsPage from './pages/Settings.page'
import ThreadPage from './pages/Thread.page'

const queryClient = new QueryClient({
    defaultOptions: {
        queries: {
            staleTime: 1000 * 60 * 5,
            retry: 1,
        },
    },
})

export default function App() {
    return (
        <QueryClientProvider client={queryClient}>
            <BrowserRouter>
                <AuthWrapper>
                    <Layout>
                        <Routes>
                            <Route path='/' element={<InboxPage />} />
                            <Route path='/search' element={<SearchPage />} />
                            <Route path='/thread/:threadId' element={<ThreadPage />} />
                            <Route path='/settings' element={<SettingsPage />} />
                        </Routes>
                    </Layout>
                </AuthWrapper>
            </BrowserRouter>
        </QueryClientProvider>
    )
}
