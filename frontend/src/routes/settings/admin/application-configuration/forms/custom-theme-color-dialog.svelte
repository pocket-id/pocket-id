<script lang="ts">
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Dialog from '#lib/components/ui/dialog/index.ts';
	import * as Field from '#lib/components/ui/field/index.ts';
	import { Input } from '#lib/components/ui/input/index.ts';
	import { Switch } from '#lib/components/ui/switch/index.ts';
	import * as Tabs from '#lib/components/ui/tabs/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import {
		colorThemeFor,
		customColorTheme,
		deriveAccent,
		parseColor,
		primaryVars
	} from '#lib/utils/color-theme-util.ts';
	import { preventDefault } from '#lib/utils/event-util.ts';
	import { LucideInfo } from '@lucide/svelte';
	import { mode } from 'mode-watcher';

	let {
		open = $bindable(false),
		initialColor,
		initialDarkSidebar = false,
		onUse
	}: {
		open: boolean;
		initialColor?: string;
		initialDarkSidebar?: boolean;
		onUse: (color: string, darkSidebar: boolean) => void;
	} = $props();

	let color = $state('');
	let darkSidebar = $state(false);
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
		const theme = colorThemeFor(customColorTheme(color.trim(), darkSidebar));
		const modeName: 'light' | 'dark' = mode.current === 'dark' ? 'dark' : 'light';
		const t = theme.tokens[modeName];
		return {
			light: deriveAccent(theme.accent, theme.accentInputs.light),
			dark: deriveAccent(theme.accent, theme.accentInputs.dark),
			current: modeName,
			sidebar: {
				background: t.sidebar,
				selected: darkSidebar ? t['sidebar-selected'] : t['fill-selected'],
				text: darkSidebar ? t['sidebar-muted'] : t['fg-muted']
			}
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
		if (!open) return;
		color = initialColor ?? '';
		darkSidebar = initialDarkSidebar;
	});

	function use() {
		showError = true;
		if (!valid) return;
		onUse(color.trim(), darkSidebar);
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

			<Field.Set>
				<Field.Legend variant="label">{m.sidebar()}</Field.Legend>
				<Tabs.Root
					bind:value={() => (darkSidebar ? 'dark' : 'light'), (v) => (darkSidebar = v === 'dark')}
				>
					<Tabs.List aria-label={m.sidebar()}>
						<Tabs.Trigger value="light">{m.light()}</Tabs.Trigger>
						<Tabs.Trigger value="dark">{m.dark()}</Tabs.Trigger>
					</Tabs.List>
				</Tabs.Root>
			</Field.Set>

			<!-- The phone bottom sheet grows upward, so the preview and notes sit above the field and the error line is reserved there to keep the field still while typing -->
			{#if fitted}
				<div class="flex flex-col gap-1.5 max-sm:-order-1" style={previewStyle}>
					<span class="text-fg-muted text-caption font-medium">{m.preview()}</span>
					<div
						class="bg-surface flex overflow-hidden rounded-md shadow-[inset_0_0_0_1px_var(--border)]"
					>
						<div
							class="flex w-20 shrink-0 flex-col gap-1.5 p-2.5"
							style="background: {fitted.sidebar.background}"
							aria-hidden="true"
						>
							<span class="h-4 rounded-[4px]" style="background: {fitted.sidebar.selected}"></span>
							<span
								class="mx-1.5 h-1.5 w-3/4 rounded-full opacity-60"
								style="background: {fitted.sidebar.text}"
							></span>
							<span
								class="mx-1.5 h-1.5 w-1/2 rounded-full opacity-60"
								style="background: {fitted.sidebar.text}"
							></span>
						</div>
						<div class="flex flex-1 flex-wrap items-center gap-3 p-4">
							<span class="pointer-events-none flex items-center gap-3" aria-hidden="true">
								<Button type="button" tabindex={-1}>{m.config_preview_button()}</Button>
								<Switch checked tabindex={-1} />
							</span>
							<span class="text-fg-muted text-mono-s ml-auto font-mono">{previewVars?.primary}</span
							>
						</div>
					</div>
				</div>
			{/if}
			<div class="flex flex-col gap-1 empty:hidden max-sm:-order-1" aria-live="polite">
				{#each notes as note (note)}
					<p class="text-caption text-fg-muted flex items-start gap-1.5">
						<LucideInfo class="mt-[3px] size-3 shrink-0" />
						{note}
					</p>
				{/each}
			</div>

			<Dialog.Footer>
				<Button type="button" variant="ghost" onclick={() => (open = false)}>{m.cancel()}</Button>
				<Button type="submit" disabled={!valid}>{m.use_color()}</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
