import path from 'path'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [
    react({
      // PDF components also run in a Web Worker, where the window-based
      // Fast Refresh runtime cannot execute. Vite still transforms their JSX.
      exclude: [
        /\/node_modules\//,
        /\/src\/lib\/pdf\//,
        /\/src\/templates\/.*pdf.*\.[jt]sx?$/i,
      ],
    }),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
