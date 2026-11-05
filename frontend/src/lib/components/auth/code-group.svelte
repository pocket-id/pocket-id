<script lang="ts">
	import * as InputOTP from '@pocket-id/ui/input-otp';
	import { cn } from '@pocket-id/ui/utils';
	import { untrack } from 'svelte';
	import { prefersReducedMotion } from 'svelte/motion';

	let {
		value = $bindable(''),
		id,
		groups,
		invalid = false,
		readonly = false,
		// Login codes are case-sensitive, device codes are not
		uppercase = false,
		onComplete,
		...restProps
	}: {
		value?: string;
		id: string;
		groups: [number, number];
		invalid?: boolean;
		readonly?: boolean;
		uppercase?: boolean;
		onComplete?: (value: string) => void;
		'aria-describedby'?: string;
		autocomplete?: 'one-time-code';
		autofocus?: boolean;
	} = $props();

	// A damped spring sampled densely, so the cells follow one continuous curve instead of easing between a few fixed stops
	const WIGGLE = Array.from({ length: 61 }, (_, i) => {
		const t = i / 60;
		return { transform: `translateX(${-9 * Math.exp(-4 * t) * Math.sin(7 * Math.PI * t)}px)` };
	});

	let root = $state<HTMLElement | null>(null);

	// Each rejected code wiggles, since the page clears the failure before every check
	$effect(() => {
		if (!invalid) return;
		untrack(() => {
			if (!prefersReducedMotion.current) root?.animate(WIGGLE, { duration: 650 });
		});
	});

	const slotClass = $derived(cn('h-13 max-w-11 max-sm:h-11', !uppercase && 'normal-case'));
</script>

<!-- Pasting anywhere fills from the first cell, without the spaces and dashes people copy along -->
<!-- bits-ui's onComplete reruns whenever readonly or aria-invalid change, so a failed check would retry forever -->
<!-- The root ends where six or eight 44px cells and the 28px dash end, so the group outlines never stretch past the last cell -->
<InputOTP.Root
	bind:ref={root}
	inputId={id}
	class={groups[0] + groups[1] === 6 ? 'max-w-73' : 'max-w-95'}
	maxlength={groups[0] + groups[1]}
	bind:value
	onValueChange={(v) => {
		value = uppercase ? v.toUpperCase() : v;
		if (value.length === groups[0] + groups[1]) onComplete?.(value);
	}}
	pasteTransformer={(text) => text.replace(/[^a-zA-Z0-9]/g, '')}
	{readonly}
	aria-invalid={invalid || undefined}
	autocapitalize={uppercase ? 'characters' : 'off'}
	spellcheck={false}
	{...restProps}
>
	{#snippet children({ cells })}
		<InputOTP.Group>
			{#each cells.slice(0, groups[0]) as cell, i (i)}
				<InputOTP.Slot {cell} class={slotClass} aria-invalid={invalid || undefined} />
			{/each}
		</InputOTP.Group>
		<InputOTP.Separator />
		<InputOTP.Group>
			{#each cells.slice(groups[0]) as cell, i (i)}
				<InputOTP.Slot {cell} class={slotClass} aria-invalid={invalid || undefined} />
			{/each}
		</InputOTP.Group>
	{/snippet}
</InputOTP.Root>
