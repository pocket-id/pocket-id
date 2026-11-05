<script lang="ts">
	import {
		previewMode,
		appearance,
		derivedPrimary,
		primaryColors,
		surfaceRoles,
		swatches,
		type SurfaceRole
	} from '#lib/components/appearance/appearance.svelte.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { cn } from '@pocket-id/ui/utils';
	import { RadioGroup } from 'bits-ui';
	import PipetteIcon from '~icons/solar/pipette-linear';
	import PreviewSection from './preview-section.svelte';

	type Swatch = { value: string; color: string; label: string };

	let { class: className, disabled = false }: { class?: string; disabled?: boolean } = $props();

	const uid = $props.id();

	const current = $derived(previewMode.current);
	const surfaces = $derived(appearance.value.surfaces[current]);
	const primary = $derived(derivedPrimary(appearance.value, current));

	// Monochrome keeps the theme's ink, which each mode draws in its own text color, so its swatch shows the mode's version
	const MONOCHROME = 'monochrome';
	const primarySwatches = $derived<Swatch[]>([
		{
			value: MONOCHROME,
			color: derivedPrimary({ ...appearance.value, primary: null }, current).fill,
			label: 'Monochrome'
		},
		...primaryColors.map((hex) => ({ value: hex, color: hex, label: hex }))
	]);

	const greys = (role: SurfaceRole): Swatch[] =>
		swatches[current][role].map((hex) => ({ value: hex, color: hex, label: hex }));
</script>

{#snippet colorRow(
	id: string,
	label: string,
	description: string | undefined,
	value: string,
	options: Swatch[],
	set: (value: string) => void
)}
	{@const custom = !options.some((option) => option.value === value)}
	<div role="group" aria-labelledby="{uid}-{id}" class="flex flex-col gap-2.5">
		<div class="flex flex-col gap-0.5">
			<p id="{uid}-{id}" class="text-control text-fg font-medium">{label}</p>
			{#if description}
				<p class="text-caption text-fg-muted">{description}</p>
			{/if}
		</div>
		<div class="flex flex-wrap items-center gap-1.5 sm:gap-2">
			<RadioGroup.Root
				aria-labelledby="{uid}-{id}"
				orientation="horizontal"
				class="flex flex-wrap items-center gap-1.5 sm:gap-2"
				bind:value={() => value, set}
			>
				{#each options as option (option.value)}
					<RadioGroup.Item
						value={option.value}
						aria-label={option.label}
						title={option.label}
						class="swatch"
						style="background: {option.color}"
					/>
				{/each}
			</RadioGroup.Root>
			<label
				class={cn('swatch text-fg-muted hover:text-fg cursor-pointer', custom && 'checked')}
				style:background={custom ? value : undefined}
				title="Custom color"
			>
				<input
					type="color"
					class="sr-only"
					aria-label="Custom {label.toLowerCase()} color"
					value={custom ? value : options[0].color}
					oninput={(e) => set(e.currentTarget.value)}
				/>
				<PipetteIcon class={cn('size-4', custom && 'opacity-0')} />
			</label>
		</div>
	</div>
{/snippet}

<PreviewSection title="Colors" class={className} {disabled}>
	<div class="roles flex flex-col gap-6">
		{#each surfaceRoles as role (role.value)}
			{@render colorRow(
				role.value,
				role.label,
				undefined,
				surfaces[role.value],
				greys(role.value),
				(v) => (appearance.value.surfaces[current][role.value] = v)
			)}
		{/each}
		<div class="flex flex-col gap-2">
			{@render colorRow(
				'primary',
				'Primary',
				'Buttons and active states',
				appearance.value.primary ?? MONOCHROME,
				primarySwatches,
				(v) => (appearance.value.primary = v === MONOCHROME ? null : v)
			)}
			{#if primary.nearSemantic}
				<p class="text-caption text-fg-muted">{m.custom_color_note_near_status()}</p>
			{/if}
		</div>
	</div>
</PreviewSection>

<style>
	.roles :global(.swatch) {
		--hairline: rgb(0 0 0 / 0.12);
		display: grid;
		place-items: center;
		width: 2rem;
		height: 2rem;
		flex-shrink: 0;
		border-radius: 999px;
		box-shadow: inset 0 0 0 1px var(--hairline);
		transition: box-shadow var(--dur-fast) ease-out;
	}

	@media (width < 40rem) {
		.roles :global(.swatch) {
			width: 1.75rem;
			height: 1.75rem;
		}
	}

	:global(.dark) .roles :global(.swatch) {
		--hairline: rgb(255 255 255 / 0.14);
	}

	.roles :global(.swatch[data-state='checked']),
	.roles :global(.swatch.checked) {
		box-shadow:
			inset 0 0 0 1px var(--hairline),
			0 0 0 2px var(--surface),
			0 0 0 3.5px var(--fg);
	}

	.roles :global(.swatch:focus-visible),
	.roles :global(.swatch:has(:focus-visible)) {
		outline: 2px solid var(--ring);
		outline-offset: 4px;
	}
</style>
