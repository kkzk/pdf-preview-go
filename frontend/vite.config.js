import {defineConfig} from 'vite'
import {svelte} from '@sveltejs/vite-plugin-svelte'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [svelte()],
  build: {
    // PDF.js is large; the bundle is loaded locally by the desktop app
    chunkSizeWarningLimit: 1500,
  },
})
