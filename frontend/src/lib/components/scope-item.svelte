<script lang="ts">
	import AdminAccessMarker from '#lib/components/admin-access-marker.svelte';
	import { Checkbox } from '@pocket-id/ui/checkbox';
	import type { Component } from 'svelte';

	let {
		icon: Icon,
		name,
		description,
		adminAccess = false,
		selected = true,
		onSelectedChange
	}: {
		icon: Component;
		name: string;
		description: string;
		// Marks a scope that can lead to full admin access
		adminAccess?: boolean;
		selected?: boolean;
		onSelectedChange?: (selected: boolean) => void;
	} = $props();

	const textId = $props.id();
</script>

{#snippet content()}
	<Icon class="text-fg-muted size-4 shrink-0" />
	<span id={textId} class="flex min-w-0 flex-1 flex-col gap-0.5 text-start wrap-anywhere">
		<span class="text-body flex items-center gap-1.5">
			{name}
			{#if adminAccess}<AdminAccessMarker />{/if}
		</span>
		<span class="text-control text-fg-muted">{description}</span>
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
