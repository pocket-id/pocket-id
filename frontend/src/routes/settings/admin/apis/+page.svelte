<script lang="ts">
	import { goto } from '$app/navigation';
	import DocsLink from '#lib/components/docs-link.svelte';
	import OidcClientAvatar from '#lib/components/oidc-client-avatar.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import AdvancedTable from '#lib/components/table/advanced-table.svelte';
	import { Badge } from '@pocket-id/ui/badge';
	import { Button } from '@pocket-id/ui/button';
	import { m } from '#lib/paraglide/messages.js';
	import ApisService from '#lib/services/apis-service.ts';
	import type {
		AdvancedTableColumn,
		CreateAdvancedTableActions
	} from '#lib/types/advanced-table.type.ts';
	import { builtInApiId, type Api } from '#lib/types/api.type.ts';
	import type { ListRequestOptions } from '#lib/types/list-request.type.ts';
	import { openOnCreateParam } from '#lib/utils/create-param-util.ts';
	import { toast } from 'svelte-sonner';
	import {
		LucideArrowRight,
		LucideCopy,
		LucideLock,
		LucidePlus,
		LucideTrash2
	} from '@lucide/svelte';
	import { confirmDeleteApi } from './api-actions';
	import ApiCreateSheet from './api-create-sheet.svelte';

	let { data } = $props();

	const apisService = new ApisService();
	const docsUrl = 'https://pocket-id.org/docs/guides/apis';

	let tableRef = $state<AdvancedTable<Api>>();
	let createOpen = $state(false);
	openOnCreateParam(() => (createOpen = true));

	const href = (api: Api) => `/settings/admin/apis/${api.id}`;

	const fetchApis = (options: ListRequestOptions) => apisService.list(options);

	const columns: AdvancedTableColumn<Api>[] = $derived([
		{ label: m.api(), column: 'name', sortable: true, cell: ApiCell },
		{ label: m.permissions(), key: 'permissions', value: (api) => api.permissions.length },
		...(data.features?.cimd.enabled
			? [
					{
						label: m.metadata_document_clients(),
						key: 'allowCimdClients',
						hidden: true,
						cell: CimdCell
					}
				]
			: []),
		{ label: m.api_id(), key: 'id', hidden: true, cell: IdCell }
	]);

	const actions: CreateAdvancedTableActions<Api> = (api) => {
		const builtIn = api.id === builtInApiId;
		return [
			{ label: m.open(), primary: true, icon: LucideArrowRight, onClick: (api) => goto(href(api)) },
			{
				label: m.copy_resource_uri(),
				icon: LucideCopy,
				onClick: async (api) => {
					await navigator.clipboard.writeText(api.resource);
					toast.success(m.resource_uri_copied());
				}
			},
			{
				label: builtIn ? m.built_in_api_cant_be_deleted() : m.delete_ellipsis(),
				icon: LucideTrash2,
				variant: 'danger',
				disabled: builtIn,
				onClick: (api) => confirmDeleteApi(api, () => tableRef?.refresh())
			}
		];
	};
</script>

<svelte:head>
	<title>{m.apis()}</title>
</svelte:head>

{#snippet ApiCell({ item }: { item: Api })}
	<div class="flex max-w-(--table-id-max) min-w-0 items-center gap-3">
		<OidcClientAvatar id={item.id} name={item.name} hasLogo={false} size={24} />
		<div class="flex min-w-0 flex-col">
			<span class="flex min-w-0 flex-wrap items-center gap-2">
				<a
					href={href(item)}
					class="text-control text-fg truncate font-medium"
					onclick={(e) => e.stopPropagation()}>{item.name}</a
				>
				{#if item.id === builtInApiId}
					<Badge><LucideLock class="size-3" />{m.built_in()}</Badge>
				{/if}
			</span>
			<span class="text-mono-s text-fg-muted truncate font-mono">{item.resource}</span>
		</div>
	</div>
{/snippet}

{#snippet CimdCell({ item }: { item: Api })}
	{#if item.allowCimdClients}
		{m.allowed()}
	{:else}
		<span class="text-fg-muted">{m.not_allowed()}</span>
	{/if}
{/snippet}

{#snippet IdCell({ item }: { item: Api })}
	<span class="text-caption font-mono">{item.id}</span>
{/snippet}

{#snippet headerDescription()}
	{m.apis_page_description()}
	<DocsLink href={docsUrl} />
{/snippet}

<div class="flex flex-col">
	<PageHeader title={m.apis()} description={headerDescription}>
		{#snippet actions()}
			<Button onclick={() => (createOpen = true)}>
				<LucidePlus />
				{m.add_api()}
			</Button>
		{/snippet}
	</PageHeader>

	<div class="mt-10">
		<!-- No default sort, so the backend puts the built-in API first until the admin picks a column -->
		<AdvancedTable
			id="api-list"
			bind:this={tableRef}
			fetchCallback={fetchApis}
			onRowClick={(api) => goto(href(api))}
			{columns}
			{actions}
		/>
	</div>
</div>

<ApiCreateSheet bind:open={createOpen} />
