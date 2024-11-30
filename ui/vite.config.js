import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { VitePWA } from 'vite-plugin-pwa'

const frontendPort = parseInt(process.env.PORT) || 4533
const backendPort = frontendPort + 100

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      manifest: manifest(),
      strategies: 'injectManifest',
      srcDir: 'src',
      filename: 'sw.js',
      devOptions: {
        enabled: true,
        type: 'module',
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
    sourcemap: true,
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/setupTests.js',
    css: true,
    reporters: ['verbose'],
    // reporters: ['default', 'hanging-process'],
    coverage: {
      reporter: ['text', 'json', 'html'],
      include: ['src/**/*'],
      exclude: [],
    },
  },
})

// PWA manifest
function manifest() {
  return {
    name: 'Navidrome',
    short_name: 'Navidrome',
    description:
      'Navidrome, an open source web-based music collection server and streamer',
    categories: ['music', 'entertainment'],
    display: 'standalone',
    start_url: './',
    background_color: 'white',
    theme_color: 'blue',
    icons: [
      {
        src: './android-chrome-192x192.png',
        sizes: '192x192',
        type: 'image/png',
      },
      {
        src: './android-chrome-512x512.png',
        sizes: '512x512',
        type: 'image/png',
      },
    ],
  }
}
