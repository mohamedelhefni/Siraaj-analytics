import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	// Dev server forwards API calls to the Go server, so the app always uses the relative /api.
	server: { proxy: { '/api': process.env.API_PROXY_TARGET || 'http://localhost:8080' } }
});
