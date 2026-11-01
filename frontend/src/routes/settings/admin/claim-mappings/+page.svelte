<script lang="ts">
	import { goto } from '$app/navigation';
	import PageHeader from '#lib/components/page-header.svelte';
	import AdvancedTable from '#lib/components/table/advanced-table.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { getLocale } from '#lib/paraglide/runtime.js';
	import OidcClaimMappingPolicyService from '#lib/services/oidc-claim-mapping-policy-service.ts';
	import type {
		AdvancedTableColumn,
		CreateAdvancedTableActions
	} from '#lib/types/advanced-table.type.ts';
	import {
		builtInClaimMappingPolicyId,
		type OidcClaimMappingPolicyMetadataDto
	} from '#lib/types/oidc-claim-mapping-policy.type.ts';
	import { openOnCreateParam } from '#lib/utils/create-param-util.ts';
	import {
		LucideArrowRight,
		LucideCopy,
		LucidePlus,
		LucideStar,
		LucideTrash2
	} from '@lucide/svelte';
	import { claimsCount, policyHref } from './claim-mapping-util';
	import { confirmDeletePolicy, duplicatePolicy, makeDefaultPolicy } from './policy-actions';
	import PolicyCreateSheet from './policy-create-sheet.svelte';
	import PolicyTags from './policy-tags.svelte';
	import PolicyTile from './policy-tile.svelte';

	const policyService = new OidcClaimMappingPolicyService();
	let tableRef = $state<AdvancedTable<OidcClaimMappingPolicyMetadataDto>>();
	let createOpen = $state(false);
	openOnCreateParam(() => (createOpen = true));

	const columns: AdvancedTableColumn<OidcClaimMappingPolicyMetadataDto>[] = [
		{ label: m.policy(), column: 'name', sortable: true, cell: NameCell },
		{ label: m.clients(), key: 'clients', cell: ClientsCell },
		{
			label: m.created(),
			column: 'createdAt',
			sortable: true,
			hideBelow: 520,
			value: (policy) =>
				new Date(policy.createdAt).toLocaleDateString(getLocale(), { dateStyle: 'medium' })
		}
	];

	const actions: CreateAdvancedTableActions<OidcClaimMappingPolicyMetadataDto> = (policy) => {
		const builtIn = policy.id === builtInClaimMappingPolicyId;
		return [
			{
				label: m.open(),
				primary: true,
				icon: LucideArrowRight,
				onClick: (p) => goto(policyHref(p.id))
			},
			{ label: m.duplicate(), icon: LucideCopy, onClick: duplicatePolicy },
			{
				label: m.make_default(),
				icon: LucideStar,
				disabled: policy.isDefault,
				onClick: async (p) => (await makeDefaultPolicy(p)) && tableRef?.refresh()
			},
			{
				label: builtIn ? m.built_in_policy_cant_be_deleted() : m.delete_ellipsis(),
				icon: LucideTrash2,
				variant: 'danger',
				disabled: builtIn,
				onClick: (p) => confirmDeletePolicy(p, () => tableRef?.refresh())
			}
		];
	};
</script>

<svelte:head>
	<title>{m.claim_mappings()}</title>
</svelte:head>

{#snippet NameCell({ item }: { item: OidcClaimMappingPolicyMetadataDto })}
	<div class="flex max-w-(--table-id-max) min-w-0 items-center gap-3">
		<PolicyTile size={24} />
		<div class="flex min-w-0 flex-col">
			<span class="flex min-w-0 flex-wrap items-center gap-2">
				<a
					href={policyHref(item.id)}
					class="text-control text-fg focus-visible:outline-ring truncate rounded-xs font-medium focus-visible:outline-2 focus-visible:outline-offset-2"
					onclick={(e) => e.stopPropagation()}>{item.name}</a
				>
				<PolicyTags policy={item} />
			</span>
			<span class="text-caption text-fg-muted truncate">{claimsCount(item.claimCount)}</span>
		</div>
	</div>
{/snippet}

{#snippet ClientsCell({ item }: { item: OidcClaimMappingPolicyMetadataDto })}
	<span class="flex flex-col">
		<span class={['tabular-nums', item.clientCount === 0 && 'text-fg-muted']}
			>{item.clientCount}</span
		>
		{#if item.isDefault}
			<span class="text-caption text-fg-muted">{m.every_client_without_a_policy()}</span>
		{/if}
	</span>
{/snippet}

<div class="flex flex-col">
	<PageHeader title={m.claim_mappings()} description={m.claim_mappings_page_description()}>
		{#snippet actions()}
			<Button onclick={() => (createOpen = true)}>
				<LucidePlus />
				{m.add_policy()}
			</Button>
		{/snippet}
	</PageHeader>

	<div class="mt-10">
		<AdvancedTable
			id="claim-mapping-policy-list"
			bind:this={tableRef}
			fetchCallback={policyService.listClaimMappingPolicies}
			defaultSort={{ column: 'name', direction: 'asc' }}
			onRowClick={(policy) => goto(policyHref(policy.id))}
			{columns}
			{actions}
		/>
	</div>
</div>

<PolicyCreateSheet bind:open={createOpen} />
