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
			'**/.svelte-kit/**',
			'frontend/build/**',
			'frontend/dist/**',
			'frontend/messages/**',
			'frontend/src/lib/paraglide/**',
			'frontend/static/**',
			'packages/ui/dist/**',
			'packages/tests/.output/**',
			'packages/tests/.report/**',
			'packages/tests/.auth/**'
		],
		overrides: [
			{
				files: ['frontend/**', 'packages/ui/**'],
				options: {
					svelte: true,
					sortTailwindcss: true
				}
			}
		]
	},
	staged: {
		'frontend/**/*': "sh -c 'vp fmt --check frontend'",
		'packages/**/*': "sh -c 'vp fmt --check packages'",
		'backend/**/*.go': `sh -c 'command -v gofmt >/dev/null || exit 0; gofmt -d "$@"' gofmt`
	},
	test: {
		exclude: ['**/node_modules/**', 'packages/tests/**'],
		passWithNoTests: true
	},
	lint: {
		jsPlugins: [{ name: 'vite-plus', specifier: 'vite-plus/oxlint-plugin' }],
		rules: { 'vite-plus/prefer-vite-plus-imports': 'error' },
		options: { typeAware: true, typeCheck: false }
	}
});
