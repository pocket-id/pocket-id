import { paraglideVitePlugin } from '@inlang/paraglide-js';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [
		sveltekit(),
		tailwindcss(),
		paraglideVitePlugin({
			project: './project.inlang',
			outdir: './src/lib/paraglide',
			cookieName: 'locale',
			strategy: ['cookie', 'preferredLanguage', 'baseLocale']
		})
	],

	server: {
		host: process.env.HOST,
		proxy: {
			'/api': {
				target: process.env.DEVELOPMENT_BACKEND_URL || 'http://localhost:1411'
			},
			'/.well-known': {
				target: process.env.DEVELOPMENT_BACKEND_URL || 'http://localhost:1411'
			},
			'/app.webmanifest': {
				target: process.env.DEVELOPMENT_BACKEND_URL || 'http://localhost:1411'
			}
		}
	}
});
