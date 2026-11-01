<script lang="ts">
	import { m } from '#lib/paraglide/messages.js';
	import {
		applyColorTheme,
		colorThemePreview,
		customColorTheme,
		parseColorTheme
	} from '#lib/utils/color-theme-util.ts';
	import { colorThemes } from '#lib/utils/color-themes.ts';
	import { cn } from '#lib/utils/style.ts';
	import { Check, Plus } from '@lucide/svelte';
	import { RadioGroup } from 'bits-ui';
	import CustomThemeColorDialog from './custom-theme-color-dialog.svelte';

	let {
		value = $bindable(),
		disabled = false,
		...restProps
	}: {
		value: string;
		disabled?: boolean;
		id?: string;
		'aria-describedby'?: string;
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
	let customValue = $state<string>();
	let custom = $derived(customValue ? parseColorTheme(customValue) : null);

	// The last custom theme stays on its tile while other themes are previewed
	$effect.pre(() => {
		if (parseColorTheme(value).id === 'custom') customValue = value;
	});

	// Picking a tile previews across the whole app at once
	// Discard in the unsaved bar resets value, which re-applies the saved theme
	$effect(() => {
		applyColorTheme(value);
	});

	function onValueChange(id: string) {
		if (id !== 'custom') value = id;
		else if (customValue) value = customValue;
		else dialogOpen = true;
	}

	// Choosing Custom again while it is selected opens the dialog to edit it
	// bits-ui runs these handlers before it selects, and Space selects on keydown without a click
	function editCustom(e: MouseEvent | KeyboardEvent) {
		if (selectedId !== 'custom') return;
		if (e instanceof KeyboardEvent && e.key !== ' ') return;
		dialogOpen = true;
	}
</script>

<div class="@container w-full">
	<RadioGroup.Root
		{...restProps}
		class="grid grid-cols-2 gap-2 @md:grid-cols-3 @3xl:grid-cols-6"
		aria-label={m.color_theme()}
		bind:value={() => selectedId, onValueChange}
		{disabled}
	>
		{#each colorThemes as theme (theme.id)}
			{@render tile(theme.id, themeNames[theme.id](), theme.id)}
		{/each}
		{@render tile('custom', m.custom(), customValue ?? null)}
	</RadioGroup.Root>
</div>

<CustomThemeColorDialog
	bind:open={dialogOpen}
	initialColor={custom?.color}
	initialDarkSidebar={custom?.darkSidebar}
	onUse={(color, darkSidebar) => (value = customColorTheme(color, darkSidebar))}
/>

{#snippet tile(id: string, name: string, previewValue: string | null)}
	<RadioGroup.Item
		value={id}
		onclick={id === 'custom' ? editCustom : undefined}
		onkeydown={id === 'custom' ? editCustom : undefined}
		class={cn(
			'bg-surface flex h-13 items-center gap-2.5 rounded-sm pr-3.5 pl-2.5 text-left shadow-[inset_0_0_0_1px_var(--border)] transition-shadow duration-(--dur-fast)',
			'data-[state=checked]:bg-surface-raised hover:shadow-[inset_0_0_0_1px_var(--border-strong)] data-[state=checked]:shadow-[inset_0_0_0_1px_var(--border-selected),var(--elevation-selected)]',
			'focus-visible:outline-ring focus-visible:outline-2 focus-visible:outline-offset-2',
			'disabled:cursor-not-allowed disabled:opacity-55 data-[state=checked]:disabled:opacity-100'
		)}
	>
		{#snippet children({ checked })}
			<!-- The swatch splits the light and dark sidebar colors diagonally, with the accent as a dot in the middle -->
			{#if previewValue}
				{@const [light, dark] = colorThemePreview(previewValue)}
				<span
					class="grid size-7 shrink-0 place-items-center rounded-full shadow-[inset_0_0_0_1px_rgb(0_0_0/0.08)]"
					style="background: linear-gradient(135deg, {light.sidebar} 50%, {dark.sidebar} 50%)"
					aria-hidden="true"
				>
					<span
						class="size-2.5 rounded-full shadow-[0_0_0_2px_var(--surface)]"
						style="background: {light.primary}"
					></span>
				</span>
			{:else}
				<span
					class="bg-fill-muted text-fg-muted grid size-7 shrink-0 place-items-center rounded-full"
					aria-hidden="true"
				>
					<Plus class="size-4" />
				</span>
			{/if}
			<span class="text-control flex-1 truncate font-medium">{name}</span>
			{#if checked}<Check class="size-4 shrink-0" />{/if}
		{/snippet}
	</RadioGroup.Item>
{/snippet}
