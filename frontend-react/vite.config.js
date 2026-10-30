import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Experimento: el mismo Front en React, al lado del de Vue. Corre en 5175 para
// compararlos juntos; /api va al mismo BFF en 8000. La Lógica pura se Importa
// desde ../frontend/src con el alias `@shared`, sin copiarla.
export default defineConfig({
  plugins: [react()],
  resolve: { alias: { '@shared': new URL('../frontend/src', import.meta.url).pathname } },
  server: {
    host: '127.0.0.1',
    port: 5175,
    strictPort: true,
    fs: { allow: ['..'] },
    // MUCHI_BFF_URL Apunta el Proxy a otro BFF —el publicado, por ejemplo— para
    // Probar sin levantar la API local.
    proxy: { '/api': { target: process.env.MUCHI_BFF_URL || 'http://127.0.0.1:8000', changeOrigin: true } },
  },
  build: { outDir: 'dist', emptyOutDir: true },
})
