import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// En Desarrollo el Front corre en 5174 y el BFF en 8000: /api se reenvía para
// que el Navegador vea un solo Origen, igual que en Producción. `@shared` es
// la Lógica sin Framework: los Componentes la Usan y los Tests la Prueban.
export default defineConfig({
  plugins: [react()],
  resolve: { alias: { '@shared': new URL('./src/shared', import.meta.url).pathname } },
  server: {
    host: '127.0.0.1',
    port: 5174,
    strictPort: true,
    // MUCHI_BFF_URL Apunta el Proxy a otro BFF —el publicado, por ejemplo— para
    // Probar sin levantar la API local.
    proxy: { '/api': { target: process.env.MUCHI_BFF_URL || 'http://127.0.0.1:8000', changeOrigin: true } },
  },
  build: { outDir: 'dist', emptyOutDir: true },
})
