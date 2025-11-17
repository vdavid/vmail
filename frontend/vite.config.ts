import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// https://vite.dev/config/
export default defineConfig({
    plugins: [
        react({
            babel: {
                plugins: [['babel-plugin-react-compiler', {}]],
            },
        }),
        tailwindcss(),
    ],
    server: {
        host: '0.0.0.0',
        port: parseInt(process.env.VITE_PORT || '7556', 10),
        strictPort: true,
        proxy: {
            '/api': {
                target: process.env.VITE_API_URL || 'http://localhost:11764',
                changeOrigin: true,
                ws: true,
            },
            '/test': {
                target: process.env.VITE_API_URL || 'http://localhost:11764',
                changeOrigin: true,
            },
        },
        // Ensure SPA routing works - serve index.html for all non-API routes
        // This is critical for client-side routing to work when navigating directly to URLs
        fs: {
            // Allow serving files from one level up to the project root
            allow: ['..'],
        },
    },
    preview: {
        host: '0.0.0.0',
        port: parseInt(process.env.VITE_PORT || '7556', 10),
        strictPort: true,
        proxy: {
            '/api': {
                target: process.env.VITE_API_URL || 'http://localhost:11764',
                changeOrigin: true,
                ws: true,
            },
            '/test': {
                target: process.env.VITE_API_URL || 'http://localhost:11764',
                changeOrigin: true,
            },
        },
    },
})
