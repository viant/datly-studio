import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { fileURLToPath } from 'node:url';

export default defineConfig({
  plugins: [react()],
  build: {
    rollupOptions: {
      input: {
        studio: fileURLToPath(new URL('./index.html', import.meta.url)),
        aclReview: fileURLToPath(new URL('./acl-review.html', import.meta.url)),
      },
    },
  },
  // Studio deliberately consumes the adjacent Forge workspace while developing.
  // package.json keeps the same relationship for an installed application.
  resolve: {
    dedupe: ['react', 'react-dom', '@blueprintjs/core'],
    alias: {
      'forge/components': fileURLToPath(new URL('../../forge/src/components/index.js', import.meta.url)),
      'forge/theme': fileURLToPath(new URL('../../forge/src/components/ThemeBoundary.jsx', import.meta.url)),
      'forge/editor': fileURLToPath(new URL('../../forge/src/components/Editor.jsx', import.meta.url)),
    },
  },
  server: {
    host: '127.0.0.1',
    fs: { allow: [fileURLToPath(new URL('.', import.meta.url)), fileURLToPath(new URL('../../forge', import.meta.url))] },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/testSetup.js'],
    include: ['src/**/*.render.test.{js,jsx}'],
  },
});
