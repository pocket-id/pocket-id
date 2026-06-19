import { svelte } from '@sveltejs/vite-plugin-svelte';
import { dirname } from 'path';
import { fileURLToPath } from 'url';
import { defineConfig } from 'vite';

const __dirname = dirname(fileURLToPath(import.meta.url));

export default defineConfig({
	root: __dirname,
	plugins: [svelte()],
	build: {
		outDir: '../static/simple/qr',
		emptyOutDir: false,
		target: 'es2015',
		rollupOptions: {
			input: './src/main.ts',
			output: {
				format: 'iife',
				entryFileNames: 'app.js',
				assetFileNames: '[name].[ext]'
			}
		}
	}
});
