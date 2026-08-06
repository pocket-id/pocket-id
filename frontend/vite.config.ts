import { paraglideVitePlugin } from '@inlang/paraglide-js';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

export default defineConfig(() => {
	return {
		plugins: [
			sveltekit(),
			tailwindcss(),
			paraglideVitePlugin({
				project: './project.inlang',
				outdir: './src/lib/paraglide',
				emitTsDeclarations: true,
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
				'/internal': {
					target: process.env.DEVELOPMENT_BACKEND_URL || 'http://localhost:1411'
				},
				'/.well-known': {
					target: process.env.DEVELOPMENT_BACKEND_URL || 'http://localhost:1411'
				},
				'^/authorize(?:\\?.*)?$': {
					target: process.env.DEVELOPMENT_BACKEND_URL || 'http://localhost:1411'
				}
			}
		}
	};
});
