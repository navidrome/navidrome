import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react-swc'
import { VitePWA } from 'vite-plugin-pwa'
import eslintPlugin from '@nabla/vite-plugin-eslint'
import fs from 'fs'
import path from 'path'

let frontendPort = 4533
let backendPort = 4633
if (process.env.PORT !== undefined) {
  frontendPort = parseInt(process.env.PORT)
  backendPort = frontendPort + 100
}

// Load manifest file
const jsonFilePath = path.resolve(__dirname, './public/manifest.webmanifest')
const manifest = JSON.parse(fs.readFileSync(jsonFilePath, 'utf-8'))

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    react(),
    eslintPlugin({ formatter: 'stylish' }),
    VitePWA({
      registerType: 'autoUpdate',
      manifest,
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
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/setupTests.js',
    css: true,
    reporters: ['verbose'],
    coverage: {
      reporter: ['text', 'json', 'html'],
      include: ['src/**/*'],
      exclude: [],
    },
  },
})
