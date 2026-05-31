<script lang="ts">
	import { openConfirmDialog } from '$lib/components/confirm-dialog';
	import AdvancedTable from '$lib/components/table/advanced-table.svelte';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import type {
		AdvancedTableColumn,
		CreateAdvancedTableActions
	} from '$lib/types/advanced-table.type';
	import type { AllAppConfig } from '$lib/types/application-configuration.type';
	import type { CustomField } from '$lib/types/custom-field.type';
	import type { ListRequestOptions } from '$lib/types/list-request.type';
	import {
		LucideCaseSensitive,
		LucideCheck,
		LucideHash,
		LucidePencil,
		LucidePlus,
		LucideToggleLeft,
		LucideTrash,
		LucideX
	} from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import AppConfigCustomFieldsDialog from './forms/app-config-custom-fields-dialog.svelte';

	let {
		appConfig,
		callback
	}: {
		appConfig: AllAppConfig;
		callback: (updatedConfig: Partial<AllAppConfig>) => Promise<void>;
	} = $props();

	let customFields = $state<CustomField[]>(appConfig.customFields);
	let selectedCustomField = $state<CustomField>();
	let showCustomFieldsDialog = $state(false);
	let table = $state<any>();

	const columns: AdvancedTableColumn<CustomField>[] = [
		{
			label: m.display_name(),
			column: 'displayName',
			sortable: true
		},
		{
			label: m.key(),
			column: 'key',
			sortable: true,
		},
		{
			label: m.type(),
			column: 'type',
			hidden: true,
			sortable: true,
			cell: TypeCell
		},
		{
			label: m.available_for(),
			column: 'target',
			sortable: true,
			cell: TargetCell
		},
		{
			label: m.user_editable(),
			column: 'userEditable',
			sortable: true,
			cell: BoolCell
		},
		{
			label: m.required(),
			column: 'required',
			sortable: true,
            hidden: true,
			cell: BoolCell
		}
	];

	const actions: CreateAdvancedTableActions<CustomField> = (_) => [
		{
			label: m.edit(),
			primary: true,
			icon: LucidePencil,
			onClick: (field) => {
				selectedCustomField = field;
				showCustomFieldsDialog = true;
			}
		},
		{
			label: m.delete(),
			icon: LucideTrash,
			variant: 'danger',
			onClick: (field) =>
				openConfirmDialog({
					title: m.delete_custom_field_name({ name: field.displayName }),
					message: m.delete_custom_field_description({ name: field.displayName }),
					confirm: {
						label: m.delete(),
						destructive: true,
						action: () => deleteCustomField(field)
					}
				})
		}
	];

	function openCreateDialog() {
		selectedCustomField = undefined;
		showCustomFieldsDialog = true;
	}

	async function updateCustomField(customField: CustomField) {
		const existingIndex = customFields.findIndex((f) => f.id === customField.id);
		const nextCustomFields = [...customFields];

		if (existingIndex !== -1) {
			nextCustomFields[existingIndex] = customField;
		} else {
			nextCustomFields.push(customField);
		}
		await callback({ customFields: nextCustomFields });
		customFields = nextCustomFields;
		await table?.refresh();
	}

	async function deleteCustomField(customField: CustomField) {
		const nextCustomFields = customFields.filter((field) => field.id !== customField.id);
		await callback({ customFields: nextCustomFields });
		customFields = nextCustomFields;
		toast.success(m.custom_fields_updated_successfully());
		await table?.refresh();
	}

	function fetchCallback(requestOptions: ListRequestOptions) {
		const data = [...customFields].sort((a, b) => {
			const column = requestOptions.sort?.column || 'displayName';
			const direction = requestOptions.sort?.direction === 'asc' ? 1 : -1;
			const aValue = String((a as Record<string, unknown>)[column] ?? '');
			const bValue = String((b as Record<string, unknown>)[column] ?? '');
			if (aValue < bValue) return -1 * direction;
			if (aValue > bValue) return 1 * direction;
			return 0;
		});

		return Promise.resolve({
			data,
			pagination: {
				currentPage: 1,
				totalPages: 1,
				itemsPerPage: data.length,
				totalItems: data.length
			}
		});
	}

	$effect(() => {
		customFields = appConfig.customFields;
	});
</script>

{#snippet TypeCell({ item }: { item: CustomField })}
	<span class="flex gap-1">
		{#if item.type === 'boolean'}
			<LucideToggleLeft class="size-4 my-auto" />
		{:else if item.type == 'number'}
			<LucideHash class="size-3 my-auto" />
		{:else}
			<LucideCaseSensitive class="size-4 my-auto" />
		{/if}
	</span>
{/snippet}

{#snippet TargetCell({ item }: { item: CustomField })}
	{#if item.target === 'both'}
		<Badge variant="secondary">{m.users_and_groups()}</Badge>
	{:else if item.target === 'group'}
		<Badge variant="outline">{m.groups()}</Badge>
	{:else}
		<Badge variant="outline">{m.users()}</Badge>
	{/if}
{/snippet}

{#snippet BoolCell({ item }: { item: CustomField })}
	{#if item.userEditable}
		<LucideCheck class="size-4" />
	{:else}
		<LucideX class="size-4 text-muted-foreground" />
	{/if}
{/snippet}

<div class="flex flex-col gap-3">
	<AdvancedTable
		bind:this={table}
		id="custom-fields-table"
		{fetchCallback}
		{actions}
		defaultSort={{ column: 'displayName', direction: 'desc' }}
		paginationDisabled
		withoutSearch
		{columns}
	/>
	<div class="flex justify-end mt-5">
		<Button onclick={openCreateDialog}>
			<LucidePlus class="mr-1 size-4" />
			{m.add_custom_field()}
		</Button>
	</div>
</div>

<AppConfigCustomFieldsDialog
	bind:show={showCustomFieldsDialog}
	existingCustomField={selectedCustomField}
	callback={updateCustomField}
/>
