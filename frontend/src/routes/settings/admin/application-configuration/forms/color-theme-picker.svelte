<script lang="ts">
	import { m } from '#lib/paraglide/messages.js';
	import {
		applyColorTheme,
		colorThemePreview,
		parseColorTheme
	} from '#lib/utils/color-theme-util.ts';
	import { colorThemes } from '#lib/utils/color-themes.ts';
	import { cn } from '#lib/utils/style.ts';
	import { Check, Palette, Plus } from '@lucide/svelte';
	import { RadioGroup } from 'bits-ui';
	import CustomThemeColorDialog from './custom-theme-color-dialog.svelte';

	let {
		value = $bindable(),
		disabled = false
	}: {
		value: string;
		disabled?: boolean;
	} = $props();

	const themeNames: Record<string, () => string> = {
		mono: m.color_theme_mono,
		mist: m.color_theme_mist,
		sage: m.color_theme_sage,
		midnight: m.color_theme_midnight,
		lavender: m.color_theme_lavender
	};

	let dialogOpen = $state(false);
	let selectedId = $derived(parseColorTheme(value).id);
	let customColor = $state<string>();

	// The last custom color stays on its tile while other themes are previewed
	$effect.pre(() => {
		const color = parseColorTheme(value).color;
		if (color) customColor = color;
	});

	// Picking a tile previews across the whole app at once
	// Discard in the unsaved bar resets value, which re-applies the saved theme
	$effect(() => {
		applyColorTheme(value);
	});

	function onValueChange(id: string) {
		if (id !== 'custom') value = id;
		else if (customColor) value = `custom:${customColor}`;
		else dialogOpen = true;
	}
</script>

<div class="flex max-w-[760px] flex-col gap-3">
	<RadioGroup.Root
		class="grid grid-cols-2 gap-3 sm:grid-cols-3"
		aria-label={m.color_theme()}
		bind:value={() => selectedId, onValueChange}
		{disabled}
	>
		{#each colorThemes as theme (theme.id)}
			{@render tile(theme.id, themeNames[theme.id](), theme.id)}
		{/each}
		{@render tile('custom', m.custom(), customColor ? `custom:${customColor}` : null)}
	</RadioGroup.Root>

	{#if selectedId === 'custom'}
		<div class="flex flex-col items-start gap-1">
			<button
				type="button"
				class="text-fg inline-flex items-center gap-1.5 text-sm font-medium underline-offset-4 hover:underline"
				onclick={() => (dialogOpen = true)}
				{disabled}
			>
				<Palette class="size-3.5" />
				{m.edit_custom_color()}
			</button>
			<p class="text-fg-muted text-caption">{m.custom_color_theme_note()}</p>
		</div>
	{/if}
</div>

<CustomThemeColorDialog
	bind:open={dialogOpen}
	initialColor={customColor}
	onUse={(color) => (value = `custom:${color}`)}
/>

{#snippet tile(id: string, name: string, previewValue: string | null)}
	<RadioGroup.Item
		value={id}
		class={cn(
			'group bg-surface flex flex-col gap-2 rounded-lg p-2 text-left shadow-[inset_0_0_0_1px_var(--border)] transition-shadow duration-(--dur-fast)',
			'hover:shadow-[inset_0_0_0_1px_var(--border-strong)] data-[state=checked]:shadow-[inset_0_0_0_2px_var(--fg)]',
			'focus-visible:outline-ring focus-visible:outline-2 focus-visible:outline-offset-2',
			'disabled:cursor-not-allowed disabled:opacity-55 data-[state=checked]:disabled:opacity-100'
		)}
	>
		{#snippet children({ checked })}
			{#if previewValue}
				<div class="grid h-[72px] grid-cols-2 overflow-hidden rounded-md" aria-hidden="true">
					{#each colorThemePreview(previewValue) as half (half.mode)}
						<div class="flex" style="background: {half.page}">
							<div class="flex w-[34%] flex-col gap-1 p-1.5" style="background: {half.sidebar}">
								<span class="h-1 w-full rounded-full" style="background: {half.line}"></span>
								<span class="h-1 w-3/4 rounded-full opacity-50" style="background: {half.text}"
								></span>
								<span class="h-1 w-2/3 rounded-full opacity-50" style="background: {half.text}"
								></span>
							</div>
							<div class="flex flex-1 flex-col gap-1 p-1.5">
								<span class="h-1 w-4/5 rounded-full opacity-60" style="background: {half.text}"
								></span>
								<span class="h-1 w-1/2 rounded-full opacity-40" style="background: {half.text}"
								></span>
								<span
									class="mt-auto h-2.5 w-full max-w-8 rounded-[3px]"
									style="background: {half.primary}"
								></span>
							</div>
						</div>
					{/each}
				</div>
			{:else}
				<div
					class="border-border-strong text-fg-muted text-caption flex h-[72px] flex-col items-center justify-center gap-1 rounded-md border border-dashed"
					aria-hidden="true"
				>
					<Plus class="size-4" />
					{m.pick_a_color()}
				</div>
			{/if}
			<span class="flex items-center justify-between gap-2 px-0.5 text-sm font-medium">
				{name}
				{#if checked}<Check class="size-4" />{/if}
			</span>
		{/snippet}
	</RadioGroup.Item>
{/snippet}
