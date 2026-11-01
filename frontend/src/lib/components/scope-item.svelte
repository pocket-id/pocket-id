<script lang="ts">
	import Checkbox from '#lib/components/ui/checkbox/checkbox.svelte';
	import type { LucideIcon } from '@lucide/svelte';

	let {
		icon: Icon,
		value,
		category,
		selected = true,
		onSelectedChange
	}: {
		icon: LucideIcon;
		// The person's real value, such as their email address, shown over the category it belongs to
		value?: string;
		category: string;
		selected?: boolean;
		onSelectedChange?: (selected: boolean) => void;
	} = $props();

	const textId = $props.id();
</script>

{#snippet content()}
	<Icon class="text-fg-muted size-4 shrink-0" />
	<span id={textId} class="flex min-w-0 flex-1 flex-col gap-0.5 text-start wrap-anywhere">
		{#if value}
			<span class="text-body">{value}</span>
			<span class="text-control text-fg-muted">{category}</span>
		{:else}
			<span class="text-body">{category}</span>
		{/if}
	</span>
{/snippet}

{#if onSelectedChange}
	<!-- The whole row is a label so clicking anywhere on it toggles the scope -->
	<label
		class="hover:bg-fill-row-hover has-focus-visible:bg-fill-row-hover active:bg-fill-hover flex min-h-13 cursor-pointer items-center gap-3 px-4 py-2.5 transition-colors duration-(--dur-fast)"
		data-testid="scope-item"
	>
		{@render content()}
		<!-- The checkbox is a button, which a wrapping label does not name -->
		<Checkbox
			aria-labelledby={textId}
			checked={selected}
			onCheckedChange={(checked: boolean) => onSelectedChange(checked)}
		/>
	</label>
{:else}
	<div class="flex min-h-13 items-center gap-3 px-4 py-2.5">
		{@render content()}
	</div>
{/if}
