<script lang="ts">
	import CopyButton from '#lib/components/copy-button.svelte';
	import { cn } from '@pocket-id/ui/utils';

	let {
		label,
		value,
		copyLabel,
		testId,
		truncate = false
	}: {
		label: string;
		value: string;
		// The copy button's accessible name, such as "Copy API key"
		copyLabel: string;
		testId?: string;
		// Keeps a value that can be looked up again, such as a signup link, on one line
		truncate?: boolean;
	} = $props();

	const id = $props.id();
</script>

<!-- One-time secrets are shown in full and wrap, because a truncated value cannot be checked before it is copied -->
<div class="flex w-full min-w-0 flex-col gap-1.5">
	<span {id} class="text-control font-semibold">{label}</span>
	<div class="flex min-w-0 items-stretch">
		<div
			role="textbox"
			aria-readonly="true"
			aria-labelledby={id}
			tabindex="0"
			data-testid={testId}
			title={truncate ? value : undefined}
			class={cn(
				'bg-surface-sunken text-mono focus-visible:outline-ring min-h-(--h-sm) min-w-0 flex-1 rounded-l-sm border border-r-0 px-3 py-1.5 font-mono focus-visible:outline-2 focus-visible:-outline-offset-2',
				truncate ? 'truncate' : 'break-all'
			)}
		>
			{value}
		</div>
		<CopyButton labeled {value} label={copyLabel} class="h-auto rounded-l-none" />
	</div>
</div>
