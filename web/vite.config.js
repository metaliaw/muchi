import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// En Desarrollo el Front corre en 5174 y el BFF en 8000: /api se reenvía para
// que el Navegador vea un solo Origen, igual que en Producción.
export default defineConfig({
  plugins: [vue()],
  server: {
    host: '127.0.0.1',
    port: 5174,
    strictPort: true,
    proxy: { '/api': { target: 'http://127.0.0.1:8000', changeOrigin: true } },
  },
  build: { outDir: 'dist', emptyOutDir: true },
})
