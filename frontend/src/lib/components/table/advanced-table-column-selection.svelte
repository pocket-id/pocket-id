<script lang="ts" generics="TData extends Record<string, any>">
	import { buttonVariants } from '$lib/components/ui/button/index.js';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import { m } from '$lib/paraglide/messages';
	import type { AdvancedTableColumn } from '$lib/types/advanced-table.type';
	import Settings2Icon from '@lucide/svelte/icons/settings-2';

	let {
		columns,
		selectedColumns = $bindable([])
	}: { columns: AdvancedTableColumn<TData>[]; selectedColumns: string[] } = $props();
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger
		class={buttonVariants({
			variant: 'outline',
			size: 'sm',
			class: 'ml-auto h-8'
		})}
	>
		<Settings2Icon />
		<span class="hidden md:flex">{m.view()}</span>
	</DropdownMenu.Trigger>
	<DropdownMenu.Content>
		<DropdownMenu.Group>
			<DropdownMenu.Label>{m.toggle_columns()}</DropdownMenu.Label>
			<DropdownMenu.Separator />
			{#each columns as column (column)}
				<DropdownMenu.CheckboxItem
					closeOnSelect={false}
					checked={selectedColumns.includes(column.column ?? column.key!)}
					onCheckedChange={(v) => {
						const key = column.column ?? column.key!;
						if (v) {
							selectedColumns = [...selectedColumns, key];
						} else {
							selectedColumns = selectedColumns.filter((c) => c !== key);
						}
					}}
				>
					{column.label}
				</DropdownMenu.CheckboxItem>
			{/each}
		</DropdownMenu.Group>
	</DropdownMenu.Content>
</DropdownMenu.Root>
