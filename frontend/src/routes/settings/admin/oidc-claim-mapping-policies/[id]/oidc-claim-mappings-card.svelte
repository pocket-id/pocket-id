<script lang="ts">
	import * as Card from '#lib/components/ui/card/index.ts';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import type {
		OidcClaimMappingDto,
		OidcClaimMappingPolicy,
		OidcClaimMappingPolicySourceType,
		OidcUserField
	} from '#lib/types/oidc-claim-mapping-policy.type.ts';
	import { LucidePencil, LucideTrash } from '@lucide/svelte';
	import AdvancedTable from '#lib/components/table/advanced-table.svelte';
	import type { AdvancedTableColumn } from '#lib/types/advanced-table.type.ts';
	import type { ListRequestOptions, Paginated } from '#lib/types/list-request.type.ts';
	import OidcClaimMappingModal from './oidc-claim-mapping-modal.svelte';
	import { z } from 'zod/v4';
	import { createForm } from '#lib/utils/form-util.ts';
	import { trackFormChanges } from '#lib/utils/unsaved-changes-util.svelte.ts';

	let {
		policy,
		onClaimMappingsSave
	}: {
		policy: OidcClaimMappingPolicy;
		onClaimMappingsSave: (policy: OidcClaimMappingPolicy) => Promise<void>;
	} = $props();

	type OidcClaimMappingRow = OidcClaimMappingDto & { id: string };

	let tableRef: AdvancedTable<OidcClaimMappingRow>;
	let editing = $state<OidcClaimMappingDto | null>(null);
	let modalOpen = $state(false);
	let editingIndex = $state(-1);
	// Bumped on every open so the modal is rebuilt: it snapshots the mapping into its form when it is
	// created, so reusing the instance would keep showing the previously edited claim
	let editingKey = $state(0);

	let claimMappings = $state<OidcClaimMappingDto[]>(policy.claimMappings ?? []);

	// Add a new empty claim mapping
	function addClaimMapping() {
		editing = {
			claimName: '',
			sourceType: 'user_field',
			sourceValue: '',
			scope: ['profile'],
			accessToken: false,
			idToken: true,
			userInfo: true
		};
		editingIndex = -1;
		editingKey++;
		modalOpen = true;
	}

	// Edit a claim mapping at index
	function editClaimMapping(item: OidcClaimMappingRow) {
		// Drop the row id, which is a table concern the mapping itself has no field for
		const { id, ...mapping } = item;
		editing = mapping;
		editingIndex = Number(id);
		editingKey++;
		modalOpen = true;
	}

	// end of edition
	async function onSaveClaimMapping(mapping: OidcClaimMappingDto, index: number) {
		modalOpen = false;
		editing = null;
		editingIndex = -1;
		await persist(
			index === -1
				? [...claimMappings, mapping]
				: claimMappings.map((entry, i) => (i === index ? mapping : entry))
		);
	}

	// Remove a claim mapping at index
	async function removeClaimMapping(index: string) {
		await persist(claimMappings.filter((_, i) => i !== Number(index)));
	}

	const formSchema = z.object({
		id: z.string().optional(),
		name: z.string(),
		isDefault: z.boolean(),
		claimMappings: z.array(
			z.object({
				claimName: z.string().min(1),
				sourceType: z.enum(['user_field', 'custom_claim', 'static']),
				sourceValue: z.string().min(1),
				scope: z
					.array(
						z.enum(['openid', 'profile', 'email', 'address', 'phone', 'offline_access', 'groups'])
					)
					.min(1),
				accessToken: z.boolean(),
				idToken: z.boolean(),
				userInfo: z.boolean()
			})
		)
	});
	const formStore = createForm(formSchema, policy);
	const { inputs } = formStore;
	trackFormChanges(() => formStore, onClaimMappingsSave, { discard: () => tableRef?.refresh() });

	// Send the updated mappings to the server, keeping the rest of the policy untouched
	async function persist(updated: OidcClaimMappingDto[]) {
		$inputs.claimMappings.value = updated;
		await tableRef?.refresh();
	}

	// Get user field source for display
	function getSourceTypeLabel(field: OidcClaimMappingPolicySourceType): string {
		const labels: Record<OidcClaimMappingPolicySourceType, string> = {
			user_field: m.user_field(),
			custom_claim: m.custom_claims(),
			static: m.static_value()
		};
		return labels[field] || field;
	}

	// Get user field label for display
	function getUserFieldLabel(field: OidcUserField): string {
		const labels: Record<OidcUserField, string> = {
			id: m.user_identifier(),
			email: m.email(),
			email_verified: m.email_verified(),
			first_name: m.first_name(),
			last_name: m.last_name(),
			full_name: m.full_name(),
			display_name: m.display_name(),
			username: m.username(),
			locale: m.locale(),
			picture: m.profile_picture(),
			groups: m.groups()
		};
		return labels[field] || field;
	}

	const columns: AdvancedTableColumn<OidcClaimMappingRow>[] = [
		{ label: m.claim_name(), column: 'claimName' },
		{ label: m.source_type(), key: 'sourceType', cell: OidcClaimMappingSourceTypeCell },
		{ label: m.source_value(), key: 'sourceValue', cell: OidcClaimMappingSourceValueCell },
		{ label: m.scope(), column: 'scope', hidden: true },
		{ label: m.access_token(), column: 'accessToken', hidden: true },
		{ label: m.id_token(), column: 'idToken', hidden: true },
		{ label: m.userinfo(), column: 'userInfo', hidden: true },
		{ label: '', key: 'actions', cell: OidcClaimMappingActionsCell, hidden: false }
	];

	async function fetchCallback(
		options: ListRequestOptions
	): Promise<Paginated<OidcClaimMappingRow>> {
		const page = options.pagination?.page || 1;
		const limit = options.pagination?.limit || 20;
		return {
			data: $inputs.claimMappings.value
				.map((entry, index) => ({ ...entry, id: String(index) }))
				.slice((page - 1) * limit, page * limit),
			pagination: {
				currentPage: page,
				itemsPerPage: limit,
				totalItems: claimMappings.length,
				totalPages: 1
			}
		};
	}
</script>

{#snippet OidcClaimMappingSourceTypeCell({ item }: { item: OidcClaimMappingRow })}
	{getSourceTypeLabel(item.sourceType as OidcClaimMappingPolicySourceType)}
{/snippet}

{#snippet OidcClaimMappingSourceValueCell({ item }: { item: OidcClaimMappingRow })}
	{#if item.sourceType === 'user_field'}
		{getUserFieldLabel(item.sourceValue as OidcUserField)}
	{:else if item.sourceType === 'custom_claim' || item.sourceType === 'static'}
		{item.sourceValue}
	{/if}
{/snippet}

{#snippet OidcClaimMappingActionsCell({ item }: { item: OidcClaimMappingRow })}
	<div class="flex justify-end gap-1">
		<Button variant="ghost" size="sm" aria-label={m.edit()} onclick={() => editClaimMapping(item)}>
			<LucidePencil class="size-4" />
		</Button>
		<Button
			disabled={item.claimName === 'sub'}
			variant="ghost"
			size="sm"
			aria-label={m.remove_claim_mapping()}
			onclick={() => removeClaimMapping(item.id)}
		>
			<LucideTrash class="size-4" />
		</Button>
	</div>
{/snippet}

<Card.Root>
	<Card.Header>
		<div class="flex flex-wrap items-center justify-between gap-4 md:flex-nowrap">
			<div>
				<Card.Title>{m.claim_mappings()}</Card.Title>
				<Card.Description>{m.claim_mappings_description()}</Card.Description>
			</div>
			<Button class="w-full md:w-auto" onclick={addClaimMapping}>{m.add_claim_mapping()}</Button>
		</div>
	</Card.Header>
	<Card.Content>
		<AdvancedTable
			id={`claim-mappings-${policy.id}`}
			bind:this={tableRef}
			withoutSearch
			{fetchCallback}
			{columns}
			onRowClick={(item) => editClaimMapping(item)}
		/>
	</Card.Content>
</Card.Root>

{#if editing}
	{#key editingKey}
		<OidcClaimMappingModal
			index={editingIndex}
			mapping={editing}
			onSave={onSaveClaimMapping}
			bind:open={modalOpen}
		/>
	{/key}
{/if}
