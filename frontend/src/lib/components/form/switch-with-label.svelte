<script lang="ts">
	import { Label } from '$lib/components/ui/label';
	import { Switch } from '$lib/components/ui/switch/index.js';

	let {
		id,
		checked = $bindable(),
		label,
		description,
		disabled = false,
		onCheckedChange
	}: {
		id: string;
		checked: boolean;
		label: string;
		description?: string;
		disabled?: boolean;
		onCheckedChange?: (checked: boolean) => void;
	} = $props();
</script>

<div class="items-top flex space-x-2">
	<Switch
		{id}
		{disabled}
		onCheckedChange={(v) => onCheckedChange && onCheckedChange(v == true)}
		bind:checked
	/>
	<div class="grid gap-1.5 leading-none">
		<Label for={id} class="mb-0 text-sm font-medium leading-none">
			{label}
		</Label>
		{#if description}
			<p class="text-muted-foreground text-[0.8rem]">
				{description}
			</p>
		{/if}
	</div>
</div>
