<script lang="ts">
	import * as InputOTP from '#lib/components/ui/input-otp/index.ts';
	import { cn } from '#lib/utils/style.ts';

	let {
		value = $bindable(''),
		id,
		groups,
		invalid = false,
		expired = false,
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
		// An expired code keeps its characters in a muted treatment, because it is not a typo
		expired?: boolean;
		readonly?: boolean;
		uppercase?: boolean;
		onComplete?: (value: string) => void;
		'aria-describedby'?: string;
		autocomplete?: 'one-time-code';
		autofocus?: boolean;
	} = $props();

	const slotClass = $derived(
		cn('h-13 max-w-11 max-lg:h-11', !uppercase && 'normal-case', expired && 'text-fg-muted')
	);
</script>

<!-- Pasting anywhere fills from the first cell, without the spaces and dashes people copy along -->
<!-- bits-ui's onComplete reruns whenever readonly or aria-invalid change, so a failed check would retry forever -->
<!-- The root ends where six or eight 44px cells and the 28px dash end, so the group outlines never stretch past the last cell -->
<InputOTP.Root
	inputId={id}
	class={groups[0] + groups[1] === 6 ? 'max-w-73' : 'max-w-95'}
	maxlength={groups[0] + groups[1]}
	bind:value
	onValueChange={(v) => {
		value = uppercase ? v.toUpperCase() : v;
		if (value.length === groups[0] + groups[1]) onComplete?.(value);
	}}
	pasteTransformer={(text) => text.replace(/[^a-zA-Z0-9]/g, '')}
	readonly={readonly || expired}
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
