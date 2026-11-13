<script lang="ts">
	import CopyButton from '#lib/components/copy-button.svelte';

	let {
		rows
	}: {
		rows: { label: string; value: string; copyLabel: string; testId?: string }[];
	} = $props();
</script>

<!-- Identifiers and URLs are shown in full and wrap, because a truncated one cannot be checked against what an app expects -->
<dl class="flex flex-col">
	{#each rows as row (row.label)}
		<div
			class="grid min-h-11 items-center gap-x-4 gap-y-0.5 py-1.5 sm:grid-cols-[140px_minmax(0,1fr)]"
		>
			<dt class="text-caption text-fg-muted">{row.label}</dt>
			<dd class="flex min-w-0 items-center gap-1">
				<span class="text-mono-s font-mono break-all" data-testid={row.testId}>{row.value}</span>
				<CopyButton value={row.value} label={row.copyLabel} />
			</dd>
		</div>
	{/each}
</dl>
