import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react-swc'
import eslintPlugin from '@nabla/vite-plugin-eslint'

// https://vitejs.dev/config/
export default defineConfig({
  // plugins: [react(), eslintPlugin({ formatter: 'stylish' })],
  plugins: [react()],
  server: {
    host: true,
    port: parseInt(process.env.PORT) || 3000, // Use the environment variable or default to 3000
  },
  base: './',
  build: {
    outDir: 'build',
  },
})
