import { paraglideVitePlugin } from '@inlang/paraglide-js';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';
import viteCompression from 'vite-plugin-compression';

export default defineConfig((mode) => {
	return {
		plugins: [
			sveltekit(),
			tailwindcss(),
			paraglideVitePlugin({
				project: './project.inlang',
				outdir: './src/lib/paraglide',
				cookieName: 'locale',
				strategy: ['cookie', 'preferredLanguage', 'baseLocale']
			}),

			// Create gzip-compressed files
			viteCompression({
				disable: mode.isPreview,
				algorithm: 'gzip',
				ext: '.gz',
				filter: /\.(js|mjs|json|css)$/i
			}),

			// Create brotli-compressed files
			viteCompression({
				disable: mode.isPreview,
				algorithm: 'brotliCompress',
				ext: '.br',
				filter: /\.(js|mjs|json|css)$/i
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
				}
			}
		}
	};
});
