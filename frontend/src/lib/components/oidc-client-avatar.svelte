<script lang="ts" module>
	// One uppercase letter, or the initials of the first two words of a multi-word name, never mixed case
	export function monogram(name: string) {
		const initials = name
			.trim()
			.split(/\s+/)
			.map((word) => word.match(/[\p{L}\p{N}]/u)?.[0].toUpperCase())
			.filter(Boolean);
		return initials.slice(0, 2).join('') || '?';
	}

	// Small tiles keep their own radius steps, because the control radii would round them into circles, and circles are people
	const sizeClasses = {
		20: 'size-5 rounded-[5px] text-[10px]',
		24: 'size-6 rounded-[6px] text-[11px]',
		32: 'size-8 rounded-[8px] text-[13px]',
		40: 'size-10 rounded-[8px] text-[17px]',
		56: 'size-14 rounded-[12px] text-[22px]'
	};
</script>

<script lang="ts">
	import { m } from '#lib/paraglide/messages.js';
	import { cachedOidcClientLogo } from '#lib/utils/cached-image-util.ts';
	import { cn } from '#lib/utils/style.ts';
	import { mode } from 'mode-watcher';

	let {
		id,
		name,
		hasLogo,
		size = 32
	}: {
		id: string;
		name: string;
		hasLogo: boolean;
		size?: keyof typeof sizeClasses;
	} = $props();

	const src = $derived(cachedOidcClientLogo.getUrl(id, mode.current === 'light'));
	const letters = $derived(monogram(name));

	// A broken logo falls back to the monogram until the logo URL changes
	let failed = $state(false);
	$effect(() => {
		void src;
		failed = false;
	});
</script>

{#if hasLogo && !failed}
	<span
		class={cn(
			'bg-surface-raised inset-ring-border inline-grid shrink-0 place-items-center overflow-hidden inset-ring',
			sizeClasses[size]
		)}
	>
		<img
			{src}
			alt={m.name_logo({ name })}
			class="size-[70%] object-contain"
			onerror={() => (failed = true)}
		/>
	</span>
{:else}
	<!-- A single narrow capital reads as a stroke in a square, so it is set larger, and two initials are set smaller so they fit -->
	<span
		aria-hidden="true"
		class={cn(
			'bg-fill-muted text-fg-muted inset-ring-border inline-grid shrink-0 place-items-center leading-none font-medium inset-ring select-none',
			sizeClasses[size]
		)}
	>
		<span
			class={cn(/^[IJ1]$/.test(letters) && 'text-[1.2em]', letters.length > 1 && 'text-[0.86em]')}
		>
			{letters}
		</span>
	</span>
{/if}
