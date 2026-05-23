<script lang="ts">
	import { Label } from '$lib/components/ui/label';
	import { Switch } from '$lib/components/ui/switch/index.js';
	import { cn } from '$lib/utils/style';

	let {
		id,
		class: className,
		checked = $bindable(),
		label,
		description,
		disabled = false,
		onCheckedChange
	}: {
		id: string;
		class?: string;
		checked: boolean;
		label: string;
		description?: string;
		disabled?: boolean;
		onCheckedChange?: (checked: boolean) => void;
	} = $props();
</script>

<div class={cn("items-top flex space-x-2", className)}>
	<Switch
		{id}
		{disabled}
		onCheckedChange={(v) => onCheckedChange && onCheckedChange(v == true)}
		bind:checked
	/>
	<div class="grid gap-1.5 leading-none">
		<Label for={id} class="mb-0 text-sm leading-none font-medium">
			{label}
		</Label>
		{#if description}
			<p class="text-muted-foreground text-[0.8rem]">
				{description}
			</p>
		{/if}
	</div>
</div>
