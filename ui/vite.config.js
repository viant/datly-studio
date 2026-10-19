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
  // Resolve the pinned Forge package installed by npm; no sibling checkout is required.
  resolve: {
    dedupe: ['react', 'react-dom', '@blueprintjs/core'],
    alias: {
      'forge/components': fileURLToPath(new URL('./node_modules/forge/src/components/index.js', import.meta.url)),
      'forge/theme': fileURLToPath(new URL('./node_modules/forge/src/components/ThemeBoundary.jsx', import.meta.url)),
      'forge/editor': fileURLToPath(new URL('./node_modules/forge/src/components/Editor.jsx', import.meta.url)),
    },
  },
  server: {
    host: '127.0.0.1',
    fs: { allow: [fileURLToPath(new URL('../internal/managementcatalog/windows', import.meta.url)), fileURLToPath(new URL('.', import.meta.url)), fileURLToPath(new URL('./node_modules/forge', import.meta.url))] },
  },
  test: {
    environment: 'jsdom',
    // Each rendered authoring test mounts large Blueprint/Forge graphs.
    // Bound concurrent jsdom workers instead of weakening assertion timeouts.
    maxWorkers: 2,
    setupFiles: ['./src/testSetup.js'],
    include: ['src/**/*.render.test.{js,jsx}'],
  },
});
