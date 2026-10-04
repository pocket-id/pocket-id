import { defineConfig } from 'vite-plus';

export default defineConfig({
	fmt: {
		useTabs: true,
		singleQuote: true,
		trailingComma: 'none',
		printWidth: 100,
		ignorePatterns: [
			'/*.json',
			'/*.md',
			'/*.toml',
			'/*.yml',
			'/*.yaml',
			'.devcontainer/**',
			'.github/**',
			'.vscode/**',
			'backend/**',
			'dist/**',
			'docker/**',
			'scripts/**',
			'frontend/.svelte-kit/**',
			'frontend/build/**',
			'frontend/dist/**',
			'frontend/messages/**',
			'frontend/src/lib/paraglide/**',
			'frontend/static/**',
			'tests/.output/**',
			'tests/.report/**',
			'tests/.auth/**'
		],
		overrides: [
			{
				files: ['frontend/**'],
				options: {
					svelte: true,
					sortTailwindcss: true
				}
			}
		]
	},
	staged: {
		'frontend/**/*': "sh -c 'vp fmt --check frontend'",
		'tests/**/*': "sh -c 'vp fmt --check tests'",
		'email-templates/**/*': "sh -c 'vp fmt --check email-templates'"
	},
	test: {
		exclude: ['**/node_modules/**', 'tests/**'],
		passWithNoTests: true
	},
	lint: {
		jsPlugins: [{ name: 'vite-plus', specifier: 'vite-plus/oxlint-plugin' }],
		rules: { 'vite-plus/prefer-vite-plus-imports': 'error' },
		options: { typeAware: true, typeCheck: false }
	}
});
