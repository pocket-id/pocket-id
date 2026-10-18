<script lang="ts">
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Dialog from '#lib/components/ui/dialog/index.ts';
	import * as Field from '#lib/components/ui/field/index.ts';
	import { Input } from '#lib/components/ui/input/index.ts';
	import { Switch } from '#lib/components/ui/switch/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import {
		colorThemeFor,
		deriveAccent,
		parseColor,
		primaryVars
	} from '#lib/utils/color-theme-util.ts';
	import { preventDefault } from '#lib/utils/event-util.ts';
	import { mode } from 'mode-watcher';

	let {
		open = $bindable(false),
		initialColor,
		onUse
	}: {
		open: boolean;
		initialColor?: string;
		onUse: (color: string) => void;
	} = $props();

	let color = $state('');
	let showError = $state(false);
	let errorTimer: ReturnType<typeof setTimeout> | undefined;

	const parsed = $derived(parseColor(color));
	const valid = $derived(!!parsed && parsed.alpha >= 1);
	const error = $derived(
		!color.trim() || valid
			? null
			: parsed
				? m.use_a_color_without_transparency()
				: m.enter_a_css_color()
	);

	// The preview fits the accent against the tinted surfaces of the custom theme, so it is exactly what people will see
	const fitted = $derived.by(() => {
		if (!valid) return null;
		const theme = colorThemeFor(`custom:${color.trim()}`);
		const modeName: 'light' | 'dark' = mode.current === 'dark' ? 'dark' : 'light';
		return {
			light: deriveAccent(theme.accent, theme.accentInputs.light),
			dark: deriveAccent(theme.accent, theme.accentInputs.dark),
			current: modeName
		};
	});
	const previewVars = $derived(fitted ? primaryVars(fitted[fitted.current]) : null);
	const previewStyle = $derived(
		previewVars
			? Object.entries(previewVars)
					.map(([k, v]) => `--${k}: ${v}`)
					.join('; ')
			: ''
	);
	const notes = $derived.by(() => {
		if (!fitted) return [];
		const out: string[] = [];
		if (fitted.light.deepened) out.push(m.custom_color_note_darkened());
		if (fitted.dark.lifted) out.push(m.custom_color_note_lightened());
		if (fitted.light.edge || fitted.dark.edge) out.push(m.custom_color_note_edge());
		if (fitted.light.nearSemantic) out.push(m.custom_color_note_near_status());
		return out;
	});

	// Errors wait for a pause in typing, never interrupting it
	$effect(() => {
		void color;
		showError = false;
		clearTimeout(errorTimer);
		errorTimer = setTimeout(() => (showError = true), 700);
		return () => clearTimeout(errorTimer);
	});

	$effect(() => {
		if (open) color = initialColor ?? '';
	});

	function use() {
		showError = true;
		if (!valid) return;
		onUse(color.trim());
		open = false;
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Content class="top-[15%] max-w-[420px] translate-y-0 sm:max-h-[80dvh]">
		<Dialog.Header>
			<Dialog.Title>{m.custom_theme_color()}</Dialog.Title>
			<Dialog.Description>{m.custom_theme_color_description()}</Dialog.Description>
		</Dialog.Header>

		<form class="flex flex-col gap-5" onsubmit={preventDefault(use)}>
			<Field.Field data-invalid={showError && error ? true : undefined}>
				<Field.Label for="custom-theme-color">{m.color()}</Field.Label>
				<Input
					id="custom-theme-color"
					class="font-mono"
					placeholder="#2f6f4e"
					autocomplete="off"
					spellcheck="false"
					bind:value={color}
					aria-invalid={showError && error ? true : undefined}
				/>
				<div class="max-sm:min-h-[18px] sm:empty:hidden">
					{#if showError && error}
						<Field.Error>{error}</Field.Error>
					{/if}
				</div>
			</Field.Field>

			{#if fitted}
				<div class="flex flex-col gap-2" style={previewStyle}>
					<span class="text-fg-muted text-caption font-medium">{m.preview()}</span>
					<div class="bg-surface-sunken flex items-center gap-4 rounded-md p-3">
						<Button type="button" tabindex={-1}>{m.authenticate()}</Button>
						<Switch checked tabindex={-1} aria-hidden="true" />
						<span class="text-fg-muted ml-auto font-mono text-xs">{previewVars?.primary}</span>
					</div>
					<p class="text-fg-muted text-caption" aria-live="polite">{notes.join(' ')}</p>
				</div>
			{/if}

			<Dialog.Footer>
				<Button type="button" variant="ghost" onclick={() => (open = false)}>{m.cancel()}</Button>
				<Button type="submit" disabled={!valid}>{m.use_color()}</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
