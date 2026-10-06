import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

const translationURL = new URL(
  process.env.LYCHEE_TRANSLATE_URL || 'http://127.0.0.1:8000/v1/translate',
)

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: { alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) } },
  server: {
    host: '127.0.0.1',
    strictPort: true,
    proxy: {
      '^/api/translate$': {
        target: translationURL.origin,
        changeOrigin: true,
        rewrite: () => translationURL.pathname + translationURL.search,
        proxyTimeout: 30_000,
      },
    },
  },
})
