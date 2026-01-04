<script lang="ts">
	import { Checkbox } from '$lib/components/ui/checkbox';
	import * as Field from '$lib/components/ui/field';

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
	<Checkbox
		{id}
		{disabled}
		onCheckedChange={(v) => onCheckedChange && onCheckedChange(v == true)}
		bind:checked
	/>
	<Field.Field class="gap-0">
		<Field.Label for={id}>{label}</Field.Label>
		{#if description}
			<Field.Description>{description}</Field.Description>
		{/if}
	</Field.Field>
</div>
