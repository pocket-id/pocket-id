<script lang="ts">
	import { goto } from '$app/navigation';
	import HashTabs from '#lib/components/hash-tabs.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import AdvancedTable from '#lib/components/table/advanced-table.svelte';
	import { Button } from '@pocket-id/ui/button';
	import * as Tabs from '@pocket-id/ui/tabs';
	import { m } from '#lib/paraglide/messages.js';
	import { getLocale } from '#lib/paraglide/runtime.js';
	import OidcClaimMappingPolicyService from '#lib/services/oidc-claim-mapping-policy-service.ts';
	import type {
		AdvancedTableColumn,
		CreateAdvancedTableActions
	} from '#lib/types/advanced-table.type.ts';
	import {
		builtInClaimMappingPolicyId,
		type CustomScope,
		type OidcClaimMappingPolicyMetadataDto
	} from '#lib/types/oidc-claim-mapping-policy.type.ts';
	import { openOnCreateParam } from '#lib/utils/create-param-util.ts';
	import { cssMs } from '#lib/utils/style.ts';
	import ArrowRightIcon from '~icons/solar/arrow-right-linear';
	import CopyIcon from '~icons/solar/copy-bold';
	import PlusIcon from '~icons/solar/add-circle-bold';
	import StarIcon from '~icons/solar/star-bold';
	import Trash2Icon from '~icons/solar/trash-bin-minimalistic-bold';
	import { fade } from 'svelte/transition';
	import { claimsCount, policyHref } from './claim-mapping-util';
	import CustomScopeSheet from './custom-scope-sheet.svelte';
	import CustomScopesTable from './custom-scopes-table.svelte';
	import { confirmDeletePolicy, duplicatePolicy, makeDefaultPolicy } from './policy-actions';
	import PolicyCreateSheet from './policy-create-sheet.svelte';
	import PolicyTags from './policy-tags.svelte';
	import PolicyTile from './policy-tile.svelte';

	const policyService = new OidcClaimMappingPolicyService();
	let tableRef = $state<AdvancedTable<OidcClaimMappingPolicyMetadataDto>>();
	let createOpen = $state(false);
	openOnCreateParam(() => (createOpen = true));

	let tab = $state('policies');
	let scopesTableRef = $state<CustomScopesTable>();
	let scopeSheetOpen = $state(false);
	let editedScope = $state<CustomScope>();

	function openScopeSheet(scope?: CustomScope) {
		editedScope = scope;
		scopeSheetOpen = true;
	}

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
				icon: ArrowRightIcon,
				onClick: (p) => goto(policyHref(p.id))
			},
			{ label: m.duplicate(), icon: CopyIcon, onClick: duplicatePolicy },
			{
				label: m.make_default(),
				icon: StarIcon,
				disabled: policy.isDefault,
				onClick: async (p) => (await makeDefaultPolicy(p)) && tableRef?.refresh()
			},
			{
				label: builtIn ? m.built_in_policy_cant_be_deleted() : m.delete_ellipsis(),
				icon: Trash2Icon,
				variant: 'danger',
				disabled: builtIn,
				onClick: (p) => confirmDeletePolicy(p, () => tableRef?.refresh())
			}
		];
	};
</script>

<!-- The descriptions of the two tabs crossfade in one grid cell, so the header keeps its place while one replaces the other -->
{#snippet description()}
	<span class="grid">
		{#key tab}
			<span
				class="col-start-1 row-start-1"
				in:fade={{ duration: cssMs('--dur-base'), delay: cssMs('--dur-fast') }}
				out:fade={{ duration: cssMs('--dur-fast') }}
			>
				{tab === 'scopes' ? m.custom_scopes_description() : m.claim_mappings_page_description()}
			</span>
		{/key}
	</span>
{/snippet}

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
	<PageHeader title={m.claim_mappings()} {description}>
		{#snippet actions()}
			{#if tab === 'scopes'}
				<Button onclick={() => openScopeSheet()}>
					<PlusIcon />
					{m.add_scope()}
				</Button>
			{:else}
				<Button onclick={() => (createOpen = true)}>
					<PlusIcon />
					{m.add_policy()}
				</Button>
			{/if}
		{/snippet}
	</PageHeader>

	<HashTabs bind:value={tab} class="mt-10 gap-6">
		<div class="shadow-[inset_0_-1px_0_var(--border)]">
			<Tabs.List variant="line" class="shadow-none" aria-label={m.claim_mappings()}>
				<Tabs.Trigger value="policies">{m.policies()}</Tabs.Trigger>
				<Tabs.Trigger value="scopes">{m.scopes()}</Tabs.Trigger>
			</Tabs.List>
		</div>

		<Tabs.Content value="policies">
			<AdvancedTable
				id="claim-mapping-policy-list"
				bind:this={tableRef}
				fetchCallback={policyService.listClaimMappingPolicies}
				defaultSort={{ column: 'name', direction: 'asc' }}
				onRowClick={(policy) => goto(policyHref(policy.id))}
				{columns}
				{actions}
			/>
		</Tabs.Content>

		<Tabs.Content value="scopes">
			<CustomScopesTable bind:this={scopesTableRef} onEdit={openScopeSheet} />
		</Tabs.Content>
	</HashTabs>
</div>

<PolicyCreateSheet bind:open={createOpen} />
<CustomScopeSheet
	bind:open={scopeSheetOpen}
	scope={editedScope}
	onSaved={() => scopesTableRef?.refresh()}
/>
