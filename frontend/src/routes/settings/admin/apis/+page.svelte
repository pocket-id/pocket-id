<script lang="ts">
	import { goto } from '$app/navigation';
	import DocsLink from '#lib/components/docs-link.svelte';
	import OidcClientAvatar from '#lib/components/oidc-client-avatar.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import EmptyList from '#lib/components/empty-list.svelte';
	import AdvancedTable from '#lib/components/table/advanced-table.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import ApisService from '#lib/services/apis-service.ts';
	import type {
		AdvancedTableColumn,
		CreateAdvancedTableActions
	} from '#lib/types/advanced-table.type.ts';
	import type { Api } from '#lib/types/api.type.ts';
	import type { ListRequestOptions } from '#lib/types/list-request.type.ts';
	import { openOnCreateParam } from '#lib/utils/create-param-util.ts';
	import { toast } from 'svelte-sonner';
	import {
		LucideArrowRight,
		LucideArrowUpRight,
		LucideCopy,
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
	let empty = $state(data.empty);

	const href = (api: Api) => `/settings/admin/apis/${api.id}`;

	async function fetchApis(options: ListRequestOptions) {
		const res = await apisService.list(options);
		empty = res.pagination.totalItems === 0 && !options.search;
		return res;
	}

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

	const actions: CreateAdvancedTableActions<Api> = () => [
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
			label: m.delete_ellipsis(),
			icon: LucideTrash2,
			variant: 'danger',
			onClick: (api) => confirmDeleteApi(api, () => tableRef?.refresh())
		}
	];
</script>

<svelte:head>
	<title>{m.apis()}</title>
</svelte:head>

{#snippet ApiCell({ item }: { item: Api })}
	<div class="flex max-w-(--table-id-max) min-w-0 items-center gap-3">
		<OidcClientAvatar id={item.id} name={item.name} hasLogo={false} size={24} />
		<div class="flex min-w-0 flex-col">
			<a
				href={href(item)}
				class="text-control text-fg truncate font-medium"
				onclick={(e) => e.stopPropagation()}>{item.name}</a
			>
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
	<!-- The first-run block states the purpose itself, so the header drops its description while the list is empty -->
	<PageHeader title={m.apis()} description={empty ? undefined : headerDescription}>
		{#snippet actions()}
			{#if !empty}
				<Button onclick={() => (createOpen = true)}>
					<LucidePlus />
					{m.add_api()}
				</Button>
			{/if}
		{/snippet}
	</PageHeader>

	{#if empty}
		<EmptyList columns={[m.api(), m.permissions()]} title={m.register_your_first_api()}>
			{#snippet description()}{m.no_apis_yet_description()}{/snippet}
			{#snippet action()}
				<Button onclick={() => (createOpen = true)}>
					<LucidePlus />
					{m.add_api()}
				</Button>
				<Button variant="link" class="gap-1" href={docsUrl} target="_blank" rel="noreferrer">
					{m.docs()}
					<LucideArrowUpRight />
				</Button>
			{/snippet}
		</EmptyList>
	{:else}
		<div class="mt-10">
			<AdvancedTable
				id="api-list"
				bind:this={tableRef}
				fetchCallback={fetchApis}
				defaultSort={{ column: 'name', direction: 'asc' }}
				onRowClick={(api) => goto(href(api))}
				{columns}
				{actions}
			/>
		</div>
	{/if}
</div>

<ApiCreateSheet bind:open={createOpen} />
