import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import Layout from './components/Layout'
import AuthWrapper from './components/AuthWrapper'
import InboxPage from './pages/Inbox.page'
import ThreadPage from './pages/Thread.page'
import SettingsPage from './pages/Settings.page'

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
                            <Route path='/thread/:threadId' element={<ThreadPage />} />
                            <Route path='/settings' element={<SettingsPage />} />
                        </Routes>
                    </Layout>
                </AuthWrapper>
            </BrowserRouter>
        </QueryClientProvider>
    )
}
