<script lang="ts">
	import FormInput from '#lib/components/form/form-input.svelte';
	import OidcClientAvatar from '#lib/components/oidc-client-avatar.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import SettingRow from '#lib/components/setting-row.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import * as Alert from '#lib/components/ui/alert/index.ts';
	import { Badge } from '#lib/components/ui/badge/index.ts';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.ts';
	import { Switch } from '#lib/components/ui/switch/index.ts';
	import * as Tabs from '#lib/components/ui/tabs/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import OidcClaimMappingPolicyService from '#lib/services/oidc-claim-mapping-policy-service.ts';
	import {
		builtInClaimMappingPolicyId,
		type OidcClaimMappingDto,
		type OidcUserField
	} from '#lib/types/oidc-claim-mapping-policy.type.ts';
	import type { OidcClientMetaData } from '#lib/types/oidc.type.ts';
	import { encodeClientIdParam } from '#lib/utils/client-id-util.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { createForm, deepCopy, deepEqual } from '#lib/utils/form-util.ts';
	import { trackUnsavedSection } from '#lib/utils/unsaved-changes-util.svelte.ts';
	import {
		LucideArrowLeft,
		LucideAtSign,
		LucideBadgeCheck,
		LucideChevronRight,
		LucideCopy,
		LucideEllipsis,
		LucideFingerprint,
		LucideImage,
		LucideInfo,
		LucideLanguages,
		LucideMail,
		LucidePlus,
		LucideQuote,
		LucideStar,
		LucideTag,
		LucideTrash2,
		LucideUserRound,
		LucideUsersRound,
		LucideX
	} from '@lucide/svelte';
	import type { Component } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod/v4';
	import PickerDialog from '../../user-groups/picker-dialog.svelte';
	import { backNavigate } from '../../users/navigate-back-util';
	import { claimsCount, clientsCount, userFieldLabel } from '../claim-mapping-util';
	import { confirmDeletePolicy, duplicatePolicy, makeDefaultPolicy } from '../policy-actions';
	import PolicyTags from '../policy-tags.svelte';
	import PolicyTile from '../policy-tile.svelte';
	import ClaimSheet from './claim-sheet.svelte';
	import ClaimsPreview from './claims-preview.svelte';

	// Draft claims carry a key, so the unsaved-changes bar can count one change per claim added, edited or removed
	type DraftMapping = OidcClaimMappingDto & { key: number };

	let { data } = $props();
	let policy = $state(data.policy);
	let clients = $state(data.clients);
	let assignedCount = $state(data.assignedCount);
	let tab = $state('claims');
	let sheetOpen = $state(false);
	let editing = $state<DraftMapping | null>(null);
	let clientPickerOpen = $state(false);
	let highlighted = $state<string | null>(null);

	const policyService = new OidcClaimMappingPolicyService();
	const backNavigation = backNavigate('/settings/admin/claim-mappings');
	const builtIn = policy.id === builtInClaimMappingPolicyId;

	const USER_FIELD_ICONS: Record<OidcUserField, Component> = {
		id: LucideFingerprint,
		email: LucideMail,
		email_verified: LucideBadgeCheck,
		first_name: LucideUserRound,
		last_name: LucideUserRound,
		full_name: LucideUserRound,
		display_name: LucideUserRound,
		username: LucideAtSign,
		locale: LucideLanguages,
		picture: LucideImage,
		groups: LucideUsersRound
	};

	let nextKey = 0;
	const withKeys = (mappings: OidcClaimMappingDto[]) =>
		mappings.map((mapping) => ({ ...mapping, key: nextKey++ }));
	let draft = $state<DraftMapping[]>(withKeys(policy.claimMappings));
	let saved = $state.raw(deepCopy($state.snapshot(draft)));

	const formStore = createForm(
		z.object({ name: z.string().min(1).max(255), isDefault: z.boolean() }),
		{
			name: policy.name,
			isDefault: policy.isDefault
		}
	);
	const { inputs } = formStore;

	const claimChanges = $derived.by(() => {
		const keys = new Set([...saved, ...draft].map((mapping) => mapping.key));
		return [...keys].filter((key) => {
			const before = saved.find((mapping) => mapping.key === key);
			const after = draft.find((mapping) => mapping.key === key);
			return !before || !after || !deepEqual(before, after);
		}).length;
	});

	const stripKeys = (mappings: DraftMapping[]) =>
		mappings.map(({ key: _key, ...mapping }) => mapping);

	trackUnsavedSection(
		() => {
			// Reading the store picks up commit(), which moves the baseline without touching the inputs
			void $inputs;
			return formStore.changedCount() + claimChanges;
		},
		async () => {
			const values = formStore.validate();
			if (!values) throw new Error(m.could_not_save_check_the_highlighted_fields());
			await policyService.updateClaimMappingPolicy(policy.id, {
				...values,
				claimMappings: stripKeys(draft)
			});
			formStore.commit(values);
			saved = deepCopy($state.snapshot(draft));
			await reloadPolicy();
		},
		() => {
			formStore.reset();
			draft = deepCopy(saved);
		},
		{ validate: () => formStore.validate() !== null }
	);

	// The counts depend on the default flag and on clients assigned elsewhere, so they are read again from the server
	async function reloadPolicy() {
		const [fresh, assigned] = await Promise.all([
			policyService.getClaimMappingPolicy(policy.id),
			policyService.listClients(policy.id, {
				pagination: { page: 1, limit: 100 },
				sort: { column: 'name', direction: 'asc' }
			})
		]);
		policy = fresh;
		clients = assigned.data;
		assignedCount = assigned.pagination.totalItems;
	}

	function openSheet(mapping: DraftMapping | null) {
		editing = mapping;
		sheetOpen = true;
	}

	function saveMapping(mapping: OidcClaimMappingDto) {
		if (editing) {
			const key = editing.key;
			draft = draft.map((d) => (d.key === key ? { ...mapping, key } : d));
		} else {
			draft = [...draft, { ...mapping, key: nextKey++ }];
		}
	}

	function removeMapping() {
		const key = editing?.key;
		draft = draft.filter((d) => d.key !== key);
	}

	async function makeDefault() {
		if (!(await makeDefaultPolicy(policy))) return;
		formStore.commit({ name: policy.name, isDefault: true });
		$inputs.isDefault.value = true;
		await reloadPolicy().catch(axiosErrorToast);
	}

	async function assignClients(clientIds: string[]) {
		await Promise.all(clientIds.map((id) => policyService.assignClient(policy.id, id)));
		await reloadPolicy();
		toast.success(m.clients_assigned());
	}

	async function unassignClient(client: OidcClientMetaData) {
		try {
			await policyService.removeClient(policy.id, client.id);
			await reloadPolicy();
			toast.success(m.name_unassigned({ name: client.name }), {
				description: m.it_uses_the_default_policy_now(),
				action: {
					label: m.undo(),
					onClick: () =>
						policyService
							.assignClient(policy.id, client.id)
							.then(reloadPolicy)
							.catch(axiosErrorToast)
				}
			});
		} catch (e) {
			axiosErrorToast(e);
		}
	}

	const unassignedCount = $derived(policy.isDefault ? policy.clientCount - assignedCount : 0);

	function sourceOf(mapping: OidcClaimMappingDto) {
		if (mapping.sourceType === 'user_field') {
			const field = mapping.sourceValue as OidcUserField;
			return {
				icon: USER_FIELD_ICONS[field],
				label: userFieldLabel(field),
				kind: m.user_field(),
				mono: false
			};
		}
		if (mapping.sourceType === 'custom_claim') {
			const all = mapping.sourceValue === '*';
			return {
				icon: LucideTag,
				label: all ? m.every_custom_claim() : mapping.sourceValue,
				kind: m.custom_claim(),
				mono: !all
			};
		}
		return {
			icon: LucideQuote,
			label: JSON.stringify(mapping.sourceValue),
			kind: m.fixed_value(),
			mono: true
		};
	}

	const tokenMarks = (mapping: OidcClaimMappingDto) => [
		{ on: mapping.idToken, short: m.id_token_short(), label: m.id_token() },
		{ on: mapping.accessToken, short: m.access_token_short(), label: m.access_token() },
		{ on: mapping.userInfo, short: m.userinfo(), label: m.userinfo() }
	];
</script>

<svelte:head>
	<title>{policy.name}</title>
</svelte:head>

{#snippet count(n: number)}
	<span class="text-fg-muted tabular-nums">{n}</span>
{/snippet}

{#snippet mappingCells(mapping: OidcClaimMappingDto)}
	{@const source = sourceOf(mapping)}
	{@const marks = tokenMarks(mapping)}
	<span class="min-w-0">
		{#if mapping.claimName === '*'}
			<span class="text-control text-fg-muted italic">{m.each_custom_claims_own_key()}</span>
		{:else}
			<span class="text-mono text-fg font-mono font-medium wrap-anywhere">{mapping.claimName}</span>
		{/if}
	</span>
	<span aria-hidden="true" class="text-fg-subtle hidden md:grid"
		><LucideArrowLeft class="size-3.5" /></span
	>
	<span class="flex min-w-0 items-center gap-2">
		<span
			aria-hidden="true"
			class="bg-fill-muted text-fg-muted grid size-7 shrink-0 place-items-center rounded-sm"
		>
			<source.icon class="size-3.5" />
		</span>
		<span class="flex min-w-0 flex-col">
			<span class={['wrap-anywhere', source.mono ? 'text-mono-s font-mono' : 'text-control']}
				>{source.label}</span
			>
			<span class="text-caption text-fg-muted">{source.kind}</span>
		</span>
	</span>
	<span class="flex flex-wrap gap-1">
		{#each mapping.scope as scope (scope)}
			<Badge variant="mono">{scope}</Badge>
		{/each}
	</span>
	<span
		class="flex gap-1"
		aria-label={m.included_in_tokens({
			tokens: marks
				.filter((mark) => mark.on)
				.map((mark) => mark.label)
				.join(', ')
		})}
	>
		{#each marks as mark (mark.label)}
			<span
				aria-hidden="true"
				class={[
					'text-micro rounded-xs px-1.5 leading-4.5 font-medium',
					mark.on
						? 'bg-fill-selected text-fg'
						: 'text-fg-subtle opacity-70 shadow-[inset_0_0_0_1px_var(--border)]'
				]}>{mark.short}</span
			>
		{/each}
	</span>
{/snippet}

<div class="flex flex-col gap-8">
	<PageHeader
		title={policy.name}
		breadcrumbs={[
			{ label: m.administration() },
			{ label: m.claim_mappings(), href: '/settings/admin/claim-mappings' }
		]}
	>
		{#snippet identity()}
			<PolicyTile size={40} />
		{/snippet}
		{#snippet tags()}
			<PolicyTags {policy} />
			<span>{claimsCount(policy.claimCount)} · {clientsCount(policy.clientCount)}</span>
		{/snippet}
		{#snippet actions()}
			<Button variant="secondary" onclick={() => duplicatePolicy(policy)}>
				<LucideCopy />
				{m.duplicate()}
			</Button>
			<DropdownMenu.Root>
				<DropdownMenu.Trigger>
					{#snippet child({ props })}
						<Button {...props} variant="outline" size="icon" aria-label={m.more_actions()}>
							<LucideEllipsis />
						</Button>
					{/snippet}
				</DropdownMenu.Trigger>
				<DropdownMenu.Content align="end">
					<DropdownMenu.Item disabled={policy.isDefault} onclick={makeDefault}>
						<LucideStar />
						{m.make_default()}
					</DropdownMenu.Item>
					<DropdownMenu.Separator />
					<DropdownMenu.Item
						variant={builtIn ? 'default' : 'destructive'}
						disabled={builtIn}
						onclick={() => confirmDeletePolicy(policy, backNavigation.leave)}
					>
						<LucideTrash2 />
						{builtIn ? m.built_in_policy_cant_be_deleted() : m.delete_policy_ellipsis()}
					</DropdownMenu.Item>
				</DropdownMenu.Content>
			</DropdownMenu.Root>
		{/snippet}
	</PageHeader>

	<Tabs.Root bind:value={tab} useHash class="gap-10">
		<div class="flex shadow-[inset_0_-1px_0_var(--border)]">
			<div class="scroll-fade-x min-w-0 [scrollbar-width:none] overflow-x-auto">
				<Tabs.List variant="line" class="shadow-none max-sm:gap-4" aria-label={m.policy_sections()}>
					<Tabs.Trigger value="claims">{m.claims()} {@render count(draft.length)}</Tabs.Trigger>
					<Tabs.Trigger value="clients">{m.clients()} {@render count(assignedCount)}</Tabs.Trigger>
					<Tabs.Trigger value="settings">{m.settings()}</Tabs.Trigger>
				</Tabs.List>
			</div>
		</div>

		<Tabs.Content value="claims">
			<div class="grid items-start gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(300px,380px)]">
				<div class="flex min-w-0 flex-col gap-4">
					{#if builtIn}
						<Alert.Root variant="info">
							<LucideInfo />
							<Alert.Title>{m.built_in_policy_cant_be_changed()}</Alert.Title>
							<Alert.Description>
								{m.built_in_policy_cant_be_changed_description()}
							</Alert.Description>
						</Alert.Root>
					{/if}

					<div class="overflow-hidden rounded-md border">
						<div
							aria-hidden="true"
							class="text-caption text-fg-muted hidden grid-cols-[minmax(160px,1.2fr)_16px_minmax(140px,1.3fr)_minmax(0,1fr)_156px_16px] gap-x-3 border-b px-4 py-2.5 font-medium md:grid"
						>
							<span>{m.claim()}</span>
							<span></span>
							<span>{m.value_from()}</span>
							<span>{m.released_with()}</span>
							<span class="col-span-2">{m.included_in()}</span>
						</div>
						<div role="list" aria-label={m.claims()} onpointerleave={() => (highlighted = null)}>
							{#each draft as mapping (mapping.key)}
								{@const rowClass =
									'grid w-full min-h-(--row) grid-cols-[minmax(0,1fr)_16px] items-center gap-x-3 gap-y-1.5 px-4 py-2.5 text-start md:grid-cols-[minmax(160px,1.2fr)_16px_minmax(140px,1.3fr)_minmax(0,1fr)_156px_16px] max-md:[&>*:not(:first-child,:last-child)]:col-start-1'}
								<div
									role="listitem"
									class="border-b last:border-b-0"
									onpointerenter={() => (highlighted = mapping.claimName)}
									onfocusin={() => (highlighted = mapping.claimName)}
								>
									{#if builtIn}
										<div class={rowClass}>{@render mappingCells(mapping)}</div>
									{:else}
										<button
											type="button"
											class={[
												rowClass,
												'hover:bg-fill-row-hover focus-visible:outline-ring cursor-pointer focus-visible:outline-2 focus-visible:-outline-offset-2'
											]}
											aria-label={m.edit_name({
												name: mapping.claimName === '*' ? m.every_custom_claim() : mapping.claimName
											})}
											onclick={() => openSheet(mapping)}
										>
											{@render mappingCells(mapping)}
											<span
												aria-hidden="true"
												class="text-fg-subtle col-start-2 row-start-1 grid md:col-start-auto md:row-start-auto"
											>
												<LucideChevronRight class="size-4" />
											</span>
										</button>
									{/if}
								</div>
							{:else}
								<div
									role="listitem"
									class="text-control text-fg-muted flex min-h-(--row-2line) flex-wrap items-center gap-x-3 px-4 py-2"
								>
									{m.policy_releases_no_claims()}
									{#if !builtIn}
										<Button variant="ghost" size="sm" onclick={() => openSheet(null)}>
											<LucidePlus />
											{m.add_claim()}
										</Button>
									{/if}
								</div>
							{/each}
						</div>
						{#if !builtIn && draft.length}
							<div class="border-t px-3 py-2">
								<Button variant="ghost" size="sm" onclick={() => openSheet(null)}>
									<LucidePlus />
									{m.add_claim()}
								</Button>
							</div>
						{/if}
					</div>
				</div>

				<ClaimsPreview mappings={stripKeys(draft)} {highlighted} />
			</div>
		</Tabs.Content>

		<Tabs.Content value="clients">
			<SettingsSection title={m.assigned_clients()} description={m.assigned_clients_description()}>
				{#snippet action()}
					<Button variant="secondary" size="sm" onclick={() => (clientPickerOpen = true)}>
						<LucidePlus />
						{m.assign_clients_ellipsis()}
					</Button>
				{/snippet}
				<ul class="border-t">
					{#each clients as client (client.id)}
						<li class="flex min-h-(--row-2line) items-center gap-3 border-b py-2 ps-4 pe-2">
							<OidcClientAvatar
								id={client.id}
								name={client.name}
								hasLogo={client.hasLogo || client.hasDarkLogo}
								size={24}
							/>
							<a
								href="/settings/admin/oidc-clients/{encodeClientIdParam(client.id)}"
								class="text-control text-fg focus-visible:outline-ring min-w-0 flex-1 truncate rounded-xs font-medium hover:underline focus-visible:outline-2 focus-visible:outline-offset-2"
								>{client.name}</a
							>
							<Button
								variant="ghost"
								size="icon-xs"
								aria-label={m.unassign_name({ name: client.name })}
								onclick={() => unassignClient(client)}
							>
								<LucideX />
							</Button>
						</li>
					{:else}
						<li
							class="text-control text-fg-muted flex min-h-(--row-2line) items-center border-b px-4"
						>
							{m.no_client_is_assigned_to_this_policy()}
						</li>
					{/each}
				</ul>
				{#if unassignedCount > 0}
					<p class="text-caption text-fg-muted mt-3 flex items-start gap-1.5">
						<LucideInfo class="mt-[3px] size-3 shrink-0" />
						{unassignedCount === 1
							? m.default_policy_also_applies_one()
							: m.default_policy_also_applies({ count: unassignedCount })}
					</p>
				{/if}
			</SettingsSection>
		</Tabs.Content>

		<Tabs.Content value="settings">
			<SettingsSection title={m.policy()}>
				<div class="flex flex-col gap-6 pt-4">
					<FormInput
						class="max-w-100"
						label={m.name()}
						description={builtIn ? m.built_in_policy_keeps_its_name() : m.policy_name_helper()}
						autocomplete="off"
						readonly={builtIn}
						bind:input={$inputs.name}
					/>
					<SettingRow
						title={m.default_policy()}
						description={policy.isDefault
							? m.default_policy_on_description()
							: m.default_policy_off_description()}
					>
						{#snippet control(props)}
							<Switch
								{...props}
								disabled={policy.isDefault}
								bind:checked={$inputs.isDefault.value}
							/>
						{/snippet}
					</SettingRow>
				</div>
			</SettingsSection>
		</Tabs.Content>
	</Tabs.Root>
</div>

<ClaimSheet
	bind:open={sheetOpen}
	mapping={editing}
	others={draft.filter((d) => d.key !== editing?.key)}
	onSave={saveMapping}
	onRemove={removeMapping}
/>

<PickerDialog
	bind:open={clientPickerOpen}
	title={m.assign_clients_to_policy({ policy: policy.name })}
	searchLabel={m.search_clients()}
	load={(search) =>
		policyService
			.listAssignableClients(policy.id, {
				search,
				pagination: { page: 1, limit: 50 },
				sort: { column: 'name', direction: 'asc' }
			})
			.then((res) => res.data)}
	row={ClientRow}
	confirmLabel={(n) =>
		n === 0 ? m.assign() : n === 1 ? m.assign_one_client() : m.assign_count_clients({ count: n })}
	onConfirm={assignClients}
/>

{#snippet ClientRow(client: OidcClientMetaData)}
	<OidcClientAvatar
		id={client.id}
		name={client.name}
		hasLogo={client.hasLogo || client.hasDarkLogo}
		size={24}
	/>
	<span class="text-control text-fg min-w-0 flex-1 truncate font-medium">{client.name}</span>
{/snippet}
