<script lang="ts">
	import { Label } from '$lib/components/ui/label/index.js';
	import * as RadioGroup from '$lib/components/ui/radio-group/index.js';
	import { applyAccentColor } from '$lib/utils/accent-color-util';
	import { Check, Plus } from '@lucide/svelte';
	import CustomColorDialog from './custom-accent-color-dialog.svelte';

	let {
		selectedColor = $bindable(),
		previousColor
	}: { selectedColor: string; previousColor: string } = $props();
	let showCustomColorDialog = $state(false);

	const accentColors = [
		{ label: 'Default', color: 'default' },
		{ label: 'Rose', color: 'oklch(0.63 0.2 15)' },
		{ label: 'Orange', color: 'oklch(0.68 0.2 50)' },
		{ label: 'Amber', color: 'oklch(0.75 0.18 80)' },
		{ label: 'Green', color: 'oklch(0.65 0.2 150)' },
		{ label: 'Teal', color: 'oklch(0.6 0.15 180)' },
		{ label: 'Blue', color: 'oklch(0.6 0.2 240)' },
		{ label: 'Purple', color: 'oklch(0.6 0.24 300)' }
	];

	// Check if current accent color is a custom color (not in predefined list)
	let isCustomColor = $derived(!accentColors.some((c) => c.color === selectedColor));
	let isPreviousColorCustom = $derived(!accentColors.some((c) => c.color === previousColor));

	function handleAccentColorChange(accentValue: string) {
		selectedColor = accentValue;
		applyAccentColor(accentValue);
	}

	function handleCustomColorApply(color: string) {
		handleAccentColorChange(color);
	}
</script>

<RadioGroup.Root
	class="flex flex-wrap gap-3"
	value={isCustomColor ? 'custom' : selectedColor}
	onValueChange={(value) => {
		if (value != 'custom') {
			handleAccentColorChange(value);
		}
	}}
>
	{#each accentColors as accent}
		{@render colorOption(accent.label, accent.color, selectedColor === accent.color)}
	{/each}
	{#if isCustomColor || isPreviousColorCustom}
		{@render colorOption('Custom', isCustomColor ? selectedColor : previousColor, isCustomColor)}
	{/if}
	{@render colorOption('Custom', 'custom', false, true)}
</RadioGroup.Root>

<CustomColorDialog bind:open={showCustomColorDialog} onApply={handleCustomColorApply} />

{#snippet colorOption(
	label: string,
	color: string,
	isSelected: boolean,
	isCustomColorSelection = false
)}
	<div class="group/item relative">
		<RadioGroup.Item id={color} value={color} class="sr-only" />
		<Label
			for={color}
			class="cursor-pointer {isCustomColorSelection ? 'group' : ''}"
			onclick={() => {
				if (isCustomColorSelection) {
					showCustomColorDialog = true;
				}
			}}
		>
			<div
				class={{
					'relative z-10 size-8 rounded-full border-2 transition-all duration-200 ease-out group-hover/item:z-20 group-hover/item:scale-110': true,
					'bg-black dark:bg-white': color === 'default'
				}}
				style={color !== 'default' ? `background-color: ${color}` : ''}
				title={label}
			>
				{#if isCustomColorSelection}
					<div
						class="bg-muted absolute inset-0 flex items-center justify-center rounded-full border-2 border-dashed border-gray-300"
					>
						<Plus class="text-muted-foreground size-4" />
					</div>
				{:else if isSelected}
					<div class="absolute inset-0 flex items-center justify-center">
						<Check class="size-4 text-white drop-shadow-sm" />
					</div>
				{/if}
			</div>
			<div
				class="text-muted-foreground group-hover/item:text-foreground bg-background absolute top-12 left-1/2 z-20 max-w-0 -translate-x-1/2 transform overflow-hidden rounded-md border px-2 py-1 text-xs whitespace-nowrap opacity-0 shadow-sm transition-all duration-300 ease-out group-hover/item:max-w-[100px] group-hover/item:opacity-100"
			>
				{label}
			</div>
		</Label>
	</div>
{/snippet}
