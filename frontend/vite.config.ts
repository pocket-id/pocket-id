import { paraglideVitePlugin } from '@inlang/paraglide-js';
import { sveltekit } from '@sveltejs/kit/vite';
import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';
import sbom from 'rollup-plugin-sbom';
import { defineConfig, type Plugin, lazyPlugins } from 'vite-plus';
import packageJson from './package.json' with { type: 'json' };

export default defineConfig(() => {
	const frontendSbom = sbom({
		includeWellKnown: false,
		outDir: 'cyclonedx',
		outFilename: 'frontend.cdx',
		saveTimestamp: false
	}) as Plugin;
	frontendSbom.applyToEnvironment = (environment) => environment.name === 'client';

	return {
		plugins: lazyPlugins(() => [
			sveltekit({
				preprocess: vitePreprocess(),
				compilerOptions: {
					warningFilter: (warning) => warning.code !== 'state_referenced_locally'
				},
				adapter: adapter({
					fallback: 'index.html',
					pages: process.env.BUILD_OUTPUT_PATH ?? '../backend/frontend/dist',
					precompress: true
				}),
				version: {
					name: packageJson.version,
					pollInterval: 0
				}
			}),
			tailwindcss(),
			frontendSbom,
			paraglideVitePlugin({
				project: './project.inlang',
				outdir: './src/lib/paraglide',
				emitTsDeclarations: true,
				cookieName: 'locale',
				strategy: ['cookie', 'preferredLanguage', 'baseLocale']
			})
		]),

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
