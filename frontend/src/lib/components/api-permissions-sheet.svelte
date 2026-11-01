<script lang="ts">
	import AdminAccessMarker from '#lib/components/admin-access-marker.svelte';
	import SettingRow from '#lib/components/setting-row.svelte';
	import * as Alert from '#lib/components/ui/alert/index.ts';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { Checkbox } from '#lib/components/ui/checkbox/index.ts';
	import * as Sheet from '#lib/components/ui/sheet/index.ts';
	import { Switch } from '#lib/components/ui/switch/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { builtInApiId, type Api, type ApiClientGrant } from '#lib/types/api.type.ts';
	import { isAdminEquivalentScope, isAdminScope, scopeLabel } from '#lib/utils/api-key-scopes.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { LucideInfo, LucideTriangleAlert } from '@lucide/svelte';
	import { untrack } from 'svelte';

	let {
		open = $bindable(),
		api,
		grant,
		implicitUserAccess = false,
		implicitUserIds = [],
		showClientAccess,
		title,
		onSave
	}: {
		open: boolean;
		api: Api;
		grant: ApiClientGrant;
		implicitUserAccess?: boolean;
		implicitUserIds?: string[];
		showClientAccess: boolean;
		title?: string;
		onSave: (grant: ApiClientGrant) => Promise<void>;
	} = $props();

	const builtIn = $derived(api.id === builtInApiId);
	const permissionName = (permission: Api['permissions'][number]) =>
		builtIn ? scopeLabel(permission.key).name : permission.name;

	let userAccess = $state(false);
	let clientAccess = $state(false);
	let userIds = $state<string[]>([]);
	let clientIds = $state<string[]>([]);
	let saving = $state(false);

	$effect(() => {
		if (!open) return;
		untrack(() => {
			userAccess = grant.userDelegatedAccess;
			clientAccess = grant.clientAccess;
			userIds = [...grant.userDelegatedPermissionIds];
			clientIds = [...grant.clientPermissionIds];
		});
	});

	const toggle = (ids: string[], id: string, checked: boolean) =>
		checked ? [...ids, id] : ids.filter((existing) => existing !== id);

	const permissionIds = $derived(api.permissions.map((permission) => permission.id));
	// Permissions granted through CIMD access stay checked, so "Select all" leaves them out
	const selectableUserIds = $derived(permissionIds.filter((id) => !implicitUserIds.includes(id)));
	// A client acting as itself has no account, so on Pocket ID's own API it can only be granted admin scopes
	const clientGrantable = (permission: Api['permissions'][number]) =>
		!builtIn || isAdminScope(permission.key);
	const selectableClientIds = $derived(api.permissions.filter(clientGrantable).map((p) => p.id));
	const grantsAdminEquivalent = $derived(
		builtIn &&
			clientAccess &&
			api.permissions.some((p) => clientIds.includes(p.id) && isAdminEquivalentScope(p.key))
	);

	async function save() {
		saving = true;
		try {
			await onSave({
				userDelegatedAccess: userAccess,
				clientAccess: showClientAccess && clientAccess,
				userDelegatedPermissionIds: userAccess ? userIds : [],
				clientPermissionIds: showClientAccess && clientAccess ? clientIds : []
			});
			open = false;
		} catch (e) {
			axiosErrorToast(e);
		} finally {
			saving = false;
		}
	}
</script>

{#snippet adminWarning()}
	<Alert.Root variant="warning">
		<LucideTriangleAlert />
		<Alert.Title>{m.client_can_act_as_an_admin()}</Alert.Title>
		<Alert.Description>{m.client_can_act_as_an_admin_description()}</Alert.Description>
	</Alert.Root>
{/snippet}

<Sheet.Root bind:open>
	<Sheet.Content
		title={title ?? api.name}
		notice={grantsAdminEquivalent ? adminWarning : undefined}
	>
		<p class="text-body text-fg-muted">
			{m.choose_what_the_client_may_ask_for()}
			{#if !showClientAccess}{m.public_clients_cant_act_as_themselves()}{/if}
		</p>

		<div>
			<SettingRow title={m.on_behalf_of_people()} description={m.on_behalf_of_people_description()}>
				{#snippet control(props)}
					<Switch
						{...props}
						checked={implicitUserAccess || userAccess}
						disabled={implicitUserAccess}
						onCheckedChange={(checked) => {
							userAccess = checked;
							if (!checked) userIds = [];
						}}
					/>
				{/snippet}
			</SettingRow>
			{#if showClientAccess}
				<SettingRow title={m.as_itself()} description={m.as_itself_description()}>
					{#snippet control(props)}
						<Switch
							{...props}
							checked={clientAccess}
							onCheckedChange={(checked) => {
								clientAccess = checked;
								if (!checked) clientIds = [];
							}}
						/>
					{/snippet}
				</SettingRow>
			{/if}
		</div>

		{#if implicitUserAccess}
			<p class="text-caption text-fg-muted flex items-start gap-1.5">
				<LucideInfo class="mt-[3px] size-3 shrink-0" />
				{m.access_granted_through_cimd_access()}
			</p>
		{/if}

		{#if api.permissions.length > 0}
			<table class="text-control w-full">
				<thead>
					<tr class="text-caption text-fg-muted border-b text-start">
						<th scope="col" class="py-2 pe-3 text-start font-medium">{m.permission()}</th>
						<th
							scope="col"
							class="w-22 py-1.5 pe-3 text-start font-medium sm:w-36 sm:whitespace-nowrap"
						>
							{m.on_behalf_of_people()}
						</th>
						{#if showClientAccess}
							<th
								scope="col"
								class="w-22 py-1.5 text-start font-medium sm:w-24 sm:whitespace-nowrap"
							>
								{m.as_itself()}
							</th>
						{/if}
					</tr>
				</thead>
				<tbody>
					{#if api.permissions.length > 1}
						{@const allUser = selectableUserIds.every((id) => userIds.includes(id))}
						{@const allClient = selectableClientIds.every((id) => clientIds.includes(id))}
						<tr class="border-b">
							<th scope="row" class="text-fg py-2.5 pe-3 text-start font-medium">
								{m.select_all()}
							</th>
							<td>
								<Checkbox
									aria-label={`${m.on_behalf_of_people()}: ${m.select_all()}`}
									checked={allUser}
									indeterminate={!allUser && selectableUserIds.some((id) => userIds.includes(id))}
									disabled={selectableUserIds.length === 0}
									onCheckedChange={(checked) => {
										userIds = checked ? selectableUserIds : [];
										if (checked) userAccess = true;
									}}
								/>
							</td>
							{#if showClientAccess}
								<td>
									<Checkbox
										aria-label={`${m.as_itself()}: ${m.select_all()}`}
										checked={allClient}
										indeterminate={!allClient && clientIds.length > 0}
										onCheckedChange={(checked) => {
											clientIds = checked ? selectableClientIds : [];
											if (checked) clientAccess = true;
										}}
									/>
								</td>
							{/if}
						</tr>
					{/if}
					{#each api.permissions as permission (permission.id)}
						{@const implicit = implicitUserIds.includes(permission.id)}
						<tr class="border-b">
							<td class="py-2 pe-3">
								<span class="text-fg flex items-center gap-1.5 font-medium wrap-anywhere">
									{permissionName(permission)}
									{#if builtIn && isAdminEquivalentScope(permission.key)}<AdminAccessMarker />{/if}
								</span>
								<span class="text-mono-s text-fg-muted block font-mono wrap-anywhere">
									{permission.key}
								</span>
							</td>
							<td>
								<Checkbox
									aria-label={`${m.on_behalf_of_people()}: ${permissionName(permission)}`}
									checked={implicit || userIds.includes(permission.id)}
									disabled={implicit}
									title={implicit ? m.granted_through_cimd_access() : undefined}
									onCheckedChange={(checked) => {
										userIds = toggle(userIds, permission.id, checked);
										if (checked) userAccess = true;
									}}
								/>
							</td>
							{#if showClientAccess}
								<td>
									<Checkbox
										aria-label={`${m.as_itself()}: ${permissionName(permission)}`}
										checked={clientIds.includes(permission.id)}
										disabled={!clientGrantable(permission)}
										title={clientGrantable(permission)
											? undefined
											: m.clients_cant_act_on_an_account_as_themselves()}
										onCheckedChange={(checked) => {
											clientIds = toggle(clientIds, permission.id, checked);
											if (checked) clientAccess = true;
										}}
									/>
								</td>
							{/if}
						</tr>
					{/each}
				</tbody>
			</table>
		{:else}
			<p class="text-caption text-fg-muted flex items-start gap-1.5">
				<LucideInfo class="mt-[3px] size-3 shrink-0" />
				{m.api_has_no_permissions_yet({ name: api.name })}
			</p>
		{/if}

		{#snippet footer()}
			<Button variant="ghost" onclick={() => (open = false)}>{m.cancel()}</Button>
			<Button isLoading={saving} onclick={save}>{m.save_access()}</Button>
		{/snippet}
	</Sheet.Content>
</Sheet.Root>
