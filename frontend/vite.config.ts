import { paraglideVitePlugin } from '@inlang/paraglide-js';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import sbom from 'rollup-plugin-sbom';
import { defineConfig, type Plugin } from 'vite';

export default defineConfig(() => {
	const frontendSbom = sbom({
		includeWellKnown: false,
		outDir: 'cyclonedx',
		outFilename: 'frontend.cdx',
		saveTimestamp: false
	}) as Plugin;
	frontendSbom.applyToEnvironment = (environment) => environment.name === 'client';

	return {
		plugins: [
			sveltekit(),
			tailwindcss(),
			frontendSbom,
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
