
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

const isHostedBuild =
  process.env.NETLIFY === 'true' || process.env.VERCEL === '1'

if (isHostedBuild && !process.env.VITE_API_URL?.trim()) {
  throw new Error(
    'Missing VITE_API_URL. Set it to your backend URL ending in /api, then redeploy.'
  )
}

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    port: 5174,
    strictPort: true,
    allowedHosts: [
      'localhost',
      '.ngrok-free.app',
      '.ngrok-free.dev',
      '.ngrok.io'
    ],
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        configure: proxy => {
          proxy.on('proxyReq', proxyReq => proxyReq.removeHeader('origin'))
        }
      }
    }
  }
})
