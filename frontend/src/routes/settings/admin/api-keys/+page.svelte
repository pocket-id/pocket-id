<script lang="ts">
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
	import DocsLink from '#lib/components/docs-link.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import EmptyList from '#lib/components/empty-list.svelte';
	import AdvancedTable from '#lib/components/table/advanced-table.svelte';
	import { Badge } from '#lib/components/ui/badge/index.ts';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { getLocale } from '#lib/paraglide/runtime.js';
	import ApiKeyService from '#lib/services/api-key-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type {
		AdvancedTableColumn,
		CreateAdvancedTableActions
	} from '#lib/types/advanced-table.type.ts';
	import type { ApiKey } from '#lib/types/api-key.type.ts';
	import type { ListRequestOptions } from '#lib/types/list-request.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { openOnCreateParam } from '#lib/utils/create-param-util.ts';
	import { LucideArrowUpRight, LucideBan, LucidePlus, LucideRefreshCcw } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import ApiKeyCreateSheet from './api-key-create-sheet.svelte';
	import RenewApiKeyDialog from './renew-api-key-dialog.svelte';

	let { data } = $props();

	const apiKeyService = new ApiKeyService();
	const docsUrl = 'https://pocket-id.org/docs/api';
	const DAY = 24 * 60 * 60 * 1000;

	let tableRef = $state<AdvancedTable<ApiKey>>();
	let createOpen = $state(false);
	openOnCreateParam(() => (createOpen = true));
	let renewing = $state<ApiKey | null>(null);
	let empty = $state(data.empty);

	const formatDate = (date: string) =>
		new Date(date).toLocaleDateString(getLocale(), { dateStyle: 'medium' });

	// Expiry reads as a status only when it needs attention; later dates are plain
	function expiryStatus(key: ApiKey) {
		const expiresAt = new Date(key.expiresAt);
		if (expiresAt.getTime() <= Date.now())
			return { variant: 'warning' as const, label: m.expired() };

		// Count calendar days, since a day across a DST change is not 24 hours long
		const days = Math.round(
			(expiresAt.setHours(0, 0, 0, 0) - new Date().setHours(0, 0, 0, 0)) / DAY
		);
		if (days === 0) return { variant: 'warning' as const, label: m.expires_today() };
		if (days === 1) return { variant: 'warning' as const, label: m.expires_tomorrow() };
		if (days <= 7)
			return { variant: 'warning' as const, label: m.expires_in_days({ count: days }) };
		if (days <= 30) return { variant: 'muted' as const, label: m.expires_in_days({ count: days }) };
		return null;
	}

	// Phones only show the key column, so its expiry and last use move into a line under the name
	function phoneMeta(key: ApiKey) {
		const expiry = expiryStatus(key)?.label ?? m.expires_date({ date: formatDate(key.expiresAt) });
		const lastUsed = key.lastUsedAt ? `${m.last_used()} ${formatDate(key.lastUsedAt)}` : undefined;
		return [expiry, lastUsed].filter(Boolean).join(' · ');
	}

	async function fetchKeys(options: ListRequestOptions) {
		const res = await apiKeyService.list(options);
		empty = res.pagination.totalItems === 0;
		return res;
	}

	const refresh = () => tableRef?.refresh();

	const columns: AdvancedTableColumn<ApiKey>[] = [
		{ label: m.key(), column: 'name', sortable: true, cell: KeyCell },
		{ label: m.expires(), column: 'expiresAt', sortable: true, cell: ExpiresCell },
		{
			label: m.last_used(),
			column: 'lastUsedAt',
			sortable: true,
			hideBelow: 760,
			cell: LastUsedCell
		},
		{
			label: m.created(),
			column: 'createdAt',
			sortable: true,
			hidden: true,
			value: (key) => formatDate(key.createdAt)
		}
	];

	// The backend renews only expired keys
	const actions: CreateAdvancedTableActions<ApiKey> = (key) => [
		{
			label: m.renew_ellipsis(),
			icon: LucideRefreshCcw,
			hidden: new Date(key.expiresAt) > new Date(),
			onClick: () => (renewing = key)
		},
		{ label: m.revoke_ellipsis(), icon: LucideBan, variant: 'danger', onClick: revoke }
	];

	function revoke(key: ApiKey) {
		openConfirmDialog({
			title: m.revoke_name_question({ name: key.name }),
			message: m.revoke_api_key_consequence(),
			confirm: {
				label: m.revoke_name_action({ name: key.name }),
				destructive: true,
				icon: LucideBan,
				action: async () => {
					try {
						await apiKeyService.revoke(key.id);
						toast.success(m.name_revoked({ name: key.name }));
						await refresh();
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}
</script>

<svelte:head>
	<title>{m.api_keys()}</title>
</svelte:head>

{#snippet KeyCell({ item }: { item: ApiKey })}
	<div class="flex max-w-(--table-id-max) min-w-0 flex-col">
		<span class="text-control text-fg truncate font-medium">{item.name}</span>
		{#if item.description}
			<span class="text-caption text-fg-muted truncate">{item.description}</span>
		{/if}
		<span class="text-caption text-fg-muted truncate sm:hidden">{phoneMeta(item)}</span>
	</div>
{/snippet}

{#snippet ExpiresCell({ item }: { item: ApiKey })}
	{@const status = expiryStatus(item)}
	{#if status}
		<Badge variant={status.variant} title={formatDate(item.expiresAt)}>{status.label}</Badge>
	{:else}
		<span class="tabular-nums">{formatDate(item.expiresAt)}</span>
	{/if}
{/snippet}

{#snippet LastUsedCell({ item }: { item: ApiKey })}
	{#if item.lastUsedAt}
		<span class="tabular-nums">{formatDate(item.lastUsedAt)}</span>
	{:else}
		<span class="text-fg-muted">{m.never()}</span>
	{/if}
{/snippet}

{#snippet headerDescription()}
	{m.api_keys_page_description()}
	<DocsLink href={docsUrl} />
{/snippet}

<div class="flex flex-col">
	<PageHeader title={m.api_keys()} description={empty ? undefined : headerDescription}>
		{#snippet actions()}
			{#if !empty}
				<Button onclick={() => (createOpen = true)}>
					<LucidePlus />
					{m.create_api_key()}
				</Button>
			{/if}
		{/snippet}
	</PageHeader>

	{#if empty}
		<EmptyList
			columns={[m.key(), m.expires()]}
			title={m.create_your_first_api_key()}
			avatar={false}
		>
			{#snippet description()}
				{m.no_api_keys_yet_description({ appName: $appConfigStore.appName })}
			{/snippet}
			{#snippet action()}
				<Button onclick={() => (createOpen = true)}>
					<LucidePlus />
					{m.create_api_key()}
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
				id="api-key-list"
				bind:this={tableRef}
				fetchCallback={fetchKeys}
				defaultSort={{ column: 'lastUsedAt', direction: 'desc' }}
				withoutSearch
				{columns}
				{actions}
			/>
		</div>
	{/if}
</div>

<ApiKeyCreateSheet
	bind:open={createOpen}
	onCreated={async () => {
		empty = false;
		await refresh();
	}}
/>
<RenewApiKeyDialog bind:apiKey={renewing} onRenewed={refresh} />
