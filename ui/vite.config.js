import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { fileURLToPath } from 'node:url';
import { createRequire } from 'node:module';
const require = createRequire(import.meta.url);

export default defineConfig({
  plugins: [react()],
  // Source-distributed Forge widgets need their CommonJS dependencies bundled
  // in development as well as in the production Rollup build.
  optimizeDeps: {
    exclude: ['forge'],
    include: [
      '@preact/signals-react', '@preact/signals-react/runtime',
      'use-sync-external-store/shim/index.js', 'fast-deep-equal', 'numeral', 'recharts',
      '@blueprintjs/icons', '@blueprintjs/datetime2', '@blueprintjs/select',
    ],
  },
  build: {
    rollupOptions: {
      input: {
        studio: fileURLToPath(new URL('./index.html', import.meta.url)),
        aclReview: fileURLToPath(new URL('./acl-review.html', import.meta.url)),
      },
    },
  },
  // Forge is a local workspace dependency while its unified runtime is unpublished.
  // Keep one React/signals identity across Forge and this host.
  resolve: {
    dedupe: ['react', 'react-dom', '@blueprintjs/core', '@preact/signals-react'],
    alias: [
      {find:/^@preact\/signals-react$/,replacement:require.resolve('@preact/signals-react')},
      {find:/^@preact\/signals-react\/runtime$/,replacement:require.resolve('@preact/signals-react/runtime')},
      {find:/^react$/,replacement:require.resolve('react')},
      {find:/^react-dom$/,replacement:require.resolve('react-dom')},
      {find:/^react-dom\/client$/,replacement:require.resolve('react-dom/client')},
      {find:'forge/components',replacement:fileURLToPath(new URL('./node_modules/forge/src/components/index.js', import.meta.url))},
      {find:'forge/theme',replacement:fileURLToPath(new URL('./node_modules/forge/src/components/ThemeBoundary.jsx', import.meta.url))},
      {find:'forge/editor',replacement:fileURLToPath(new URL('./node_modules/forge/src/components/Editor.jsx', import.meta.url))},
    ],
  },
  server: {
    host: '127.0.0.1',
    fs: { allow: [fileURLToPath(new URL('../internal/managementcatalog/windows', import.meta.url)), fileURLToPath(new URL('.', import.meta.url)), fileURLToPath(new URL('./node_modules/forge', import.meta.url)), fileURLToPath(new URL('../../forge', import.meta.url))] },
  },
  test: {
    environment: 'jsdom',
    server: {deps: {inline: [/@preact\/signals/, /@phosphor-icons\/react/, /@uiw\/react-codemirror/]}},
    // Each rendered authoring test mounts large Blueprint/Forge graphs.
    // Bound concurrent jsdom workers instead of weakening assertion timeouts.
    maxWorkers: 2,
    setupFiles: ['./src/testSetup.js'],
    include: ['src/**/*.render.test.{js,jsx}'],
  },
});
