<script lang="ts">
	import CopyButton from '#lib/components/copy-button.svelte';
	import FormInput from '#lib/components/form/form-input.svelte';
	import OidcClientAvatar from '#lib/components/oidc-client-avatar.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import { Button } from '@pocket-id/ui/button';
	import * as DropdownMenu from '@pocket-id/ui/dropdown-menu';
	import HashTabs from '#lib/components/hash-tabs.svelte';
	import * as Tabs from '@pocket-id/ui/tabs';
	import { m } from '#lib/paraglide/messages.js';
	import ApisService from '#lib/services/apis-service.ts';
	import { Badge } from '@pocket-id/ui/badge';
	import {
		builtInApiId,
		type ApiCimdAccessUpdate,
		type ApiPermissionInput
	} from '#lib/types/api.type.ts';
	import { scopeLabel } from '#lib/utils/api-key-scopes.ts';
	import { createForm } from '#lib/utils/form-util.ts';
	import { trackFormChanges, trackUnsavedValue } from '#lib/utils/unsaved-changes-util.svelte.ts';
	import CopyIcon from '~icons/solar/copy-bold';
	import EllipsisIcon from '~icons/solar/menu-dots-bold';
	import InfoIcon from '~icons/solar/info-circle-bold-duotone';
	import LockIcon from '~icons/solar/lock-bold-duotone';
	import Trash2Icon from '~icons/solar/trash-bin-minimalistic-bold';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod/v4';
	import { backNavigate } from '../../users/navigate-back-util';
	import { confirmDeleteApi } from '../api-actions';
	import ApiCimdAccessTab from './api-cimd-access-tab.svelte';
	import ApiClientsTab from './api-clients-tab.svelte';
	import ApiPermissionsInput from './api-permissions-input.svelte';

	let { data } = $props();
	let api = $state(data.api);
	let clients = $state(data.clients);
	let permissions = $state<ApiPermissionInput[]>(toPermissionInputs(data.api.permissions));
	let clientsTab = $state<ApiClientsTab>();
	let tab = $state('general');
	// The built-in API follows Pocket ID's scopes, so admins only manage which clients may use it
	const builtIn = data.api.id === builtInApiId;

	const apisService = new ApisService();
	const backNavigation = backNavigate('/settings/admin/apis');

	function toPermissionInputs(
		apiPermissions: { key: string; name: string; description?: string }[]
	): ApiPermissionInput[] {
		return apiPermissions.map((p) => ({
			key: p.key,
			name: p.name,
			description: p.description ?? ''
		}));
	}

	const isEmptyPermission = (p: ApiPermissionInput) =>
		!p.key.trim() && !p.name.trim() && !p.description.trim();

	const formStore = createForm(z.object({ name: z.string().min(1).max(50) }), { name: api.name });
	const { inputs } = formStore;
	trackFormChanges(
		() => formStore,
		async ({ name }) => {
			api = { ...api, ...(await apisService.update(api.id, { name })) };
		}
	);

	trackUnsavedValue(
		() => permissions.filter((p) => !isEmptyPermission(p)),
		(saved) => (permissions = toPermissionInputs(saved)),
		async (updated) => {
			try {
				api = await apisService.updatePermissions(api.id, updated);
				permissions = updated;
			} finally {
				// A removed permission takes the client grants that referenced it with it
				await clientsTab?.refresh();
			}
		}
	);

	async function updateCimdAccess(update: ApiCimdAccessUpdate) {
		try {
			api = await apisService.updateCimdAccess(api.id, update);
		} finally {
			await clientsTab?.refresh();
		}
	}

	async function copy(value: string, message: string) {
		await navigator.clipboard.writeText(value);
		toast.success(message);
	}
</script>

<svelte:head>
	<title>{api.name}</title>
</svelte:head>

{#snippet count(n: number)}
	<span class="text-fg-muted tabular-nums">{n}</span>
{/snippet}

<div class="flex flex-col gap-8">
	<PageHeader
		title={api.name}
		breadcrumbs={[{ label: m.administration() }, { label: m.apis(), href: '/settings/admin/apis' }]}
	>
		{#snippet identity()}
			<OidcClientAvatar id={api.id} name={api.name} hasLogo={false} size={40} />
		{/snippet}
		{#snippet tags()}
			{#if builtIn}
				<Badge><LockIcon class="size-3" />{m.built_in()}</Badge>
			{/if}
			<span class="text-mono-s font-mono break-all">{api.resource}</span>
		{/snippet}
		{#snippet actions()}
			<DropdownMenu.Root>
				<DropdownMenu.Trigger>
					{#snippet child({ props })}
						<Button {...props} variant="outline" size="icon" aria-label={m.more_actions()}>
							<EllipsisIcon />
						</Button>
					{/snippet}
				</DropdownMenu.Trigger>
				<DropdownMenu.Content align="end">
					<DropdownMenu.Item onclick={() => copy(api.resource, m.resource_uri_copied())}>
						<CopyIcon />
						{m.copy_resource_uri()}
					</DropdownMenu.Item>
					<DropdownMenu.Item onclick={() => copy(api.id, m.api_id_copied())}>
						<CopyIcon />
						{m.copy_api_id()}
					</DropdownMenu.Item>
					<DropdownMenu.Separator />
					<DropdownMenu.Item
						variant="destructive"
						disabled={builtIn}
						onclick={() => confirmDeleteApi(api, backNavigation.leave)}
					>
						<Trash2Icon />
						{builtIn ? m.built_in_api_cant_be_deleted() : m.delete_api_ellipsis()}
					</DropdownMenu.Item>
				</DropdownMenu.Content>
			</DropdownMenu.Root>
		{/snippet}
	</PageHeader>

	<HashTabs bind:value={tab} class="gap-10">
		<div class="shadow-[inset_0_-1px_0_var(--border)]">
			<div class="scroll-fade-x [scrollbar-width:none] overflow-x-auto">
				<Tabs.List variant="line" class="shadow-none max-sm:gap-4" aria-label={m.api_sections()}>
					<Tabs.Trigger value="general">{m.general()}</Tabs.Trigger>
					<Tabs.Trigger value="permissions">
						{m.permissions()}
						{@render count(api.permissions.length)}
					</Tabs.Trigger>
					<Tabs.Trigger value="clients">{m.clients()} {@render count(clients.length)}</Tabs.Trigger>
					{#if data.features?.cimd.enabled}
						<Tabs.Trigger value="metadata-documents">{m.metadata_documents()}</Tabs.Trigger>
					{/if}
				</Tabs.List>
			</div>
		</div>

		<Tabs.Content value="general">
			<SettingsSection title={m.name_and_identifiers()}>
				<div class="flex flex-col gap-6 pt-4">
					<FormInput
						class="max-w-100"
						label={m.name()}
						description={builtIn ? m.built_in_api_keeps_its_name() : undefined}
						readonly={builtIn}
						bind:input={$inputs.name}
					/>
					<div class="flex flex-col gap-1">
						<p class="text-caption text-fg-muted mb-1 font-medium">{m.identifiers()}</p>
						<dl class="text-control flex flex-col">
							{#each [{ label: m.api_resource_uri(), value: api.resource }, { label: m.api_id(), value: api.id }] as identifier (identifier.label)}
								<div
									class="flex min-h-9 items-center gap-4 max-sm:flex-col max-sm:items-stretch max-sm:gap-0"
								>
									<dt class="text-fg-muted w-35 shrink-0">{identifier.label}</dt>
									<dd class="flex min-w-0 flex-1 items-center gap-2">
										<span class="text-mono-s min-w-0 truncate font-mono" title={identifier.value}
											>{identifier.value}</span
										>
										<CopyButton
											value={identifier.value}
											label={m.copy_name({ name: identifier.label })}
										/>
									</dd>
								</div>
							{/each}
						</dl>
						<p class="text-caption text-fg-muted flex items-start gap-1.5">
							<InfoIcon class="mt-[3px] size-3 shrink-0" />
							{m.resource_uris_cant_be_changed()}
						</p>
					</div>
				</div>
			</SettingsSection>
		</Tabs.Content>

		<Tabs.Content value="permissions">
			{#if builtIn}
				<SettingsSection
					title={m.permissions()}
					description={m.built_in_api_permissions_description()}
				>
					<ul class="border-t">
						{#each api.permissions as permission (permission.id)}
							{@const label = scopeLabel(permission.key)}
							<li class="flex flex-col gap-0.5 border-b px-4 py-2.5">
								<span class="text-control text-fg font-medium">{label.name}</span>
								<span class="text-mono-s text-fg-muted font-mono">{permission.key}</span>
								{#if label.description}
									<span class="text-caption text-fg-muted">{label.description}</span>
								{/if}
							</li>
						{/each}
					</ul>
				</SettingsSection>
			{:else}
				<SettingsSection title={m.permissions()} description={m.api_permissions_description()}>
					<div class="pt-2">
						<ApiPermissionsInput bind:permissions />
					</div>
				</SettingsSection>
			{/if}
		</Tabs.Content>

		<Tabs.Content value="clients">
			<ApiClientsTab bind:this={clientsTab} {api} bind:clients />
		</Tabs.Content>

		<Tabs.Content value="metadata-documents">
			<ApiCimdAccessTab {api} onSave={updateCimdAccess} />
		</Tabs.Content>
	</HashTabs>
</div>
