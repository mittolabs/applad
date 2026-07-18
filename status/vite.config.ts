import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 4100,
    // Dev-only: proxy the status API to the running backend so `npm run dev`
    // works standalone. In the container, nginx proxies /v1 → api instead.
    proxy: { '/v1': 'http://localhost:8080' },
  },
});
