import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react-swc'
import { VitePWA } from 'vite-plugin-pwa'
import eslintPlugin from '@nabla/vite-plugin-eslint'

let frontendPort = 4533
let backendPort = 4633
if (process.env.PORT !== undefined) {
  frontendPort = parseInt(process.env.PORT)
  backendPort = frontendPort + 100
}

// https://vitejs.dev/config/
export default defineConfig({
  // plugins: [react(), eslintPlugin({ formatter: 'stylish' })],
  plugins: [
    react(),
    VitePWA({
      registerType: 'autoUpdate',
      workbox: {
        // Workbox options
      },
    }),
  ],
  server: {
    host: true,
    port: frontendPort,
    proxy: {
      '^/(auth|api|rest|backgrounds)/.*': 'http://localhost:' + backendPort,
    },
  },
  base: './',
  build: {
    outDir: 'build',
  },
})
