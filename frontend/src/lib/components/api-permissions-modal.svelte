<script lang="ts">
	import SettingRow from '#lib/components/setting-row.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { Checkbox } from '#lib/components/ui/checkbox/index.ts';
	import * as Dialog from '#lib/components/ui/dialog/index.ts';
	import { Switch } from '#lib/components/ui/switch/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import type { Api, ApiClientGrant } from '#lib/types/api.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { LucideInfo } from '@lucide/svelte';
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

<Dialog.Root bind:open>
	<!-- The scroll padding matches the sticky footer's height, so keyboard focus never lands under it -->
	<Dialog.Content class="max-w-(--dialog-lg) scroll-pb-20 max-sm:scroll-pb-32">
		<Dialog.Header>
			<Dialog.Title>{title ?? api.name}</Dialog.Title>
			<Dialog.Description>
				{m.choose_what_the_client_may_ask_for()}
				{#if !showClientAccess}{m.public_clients_cant_act_as_themselves()}{/if}
			</Dialog.Description>
		</Dialog.Header>

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
					{#each api.permissions as permission (permission.id)}
						{@const implicit = implicitUserIds.includes(permission.id)}
						<tr class="border-b">
							<td class="py-2 pe-3">
								<span class="text-fg block font-medium wrap-anywhere">{permission.name}</span>
								<span class="text-mono-s text-fg-muted block font-mono wrap-anywhere">
									{permission.key}
								</span>
							</td>
							<td>
								<Checkbox
									aria-label={`${m.on_behalf_of_people()}: ${permission.name}`}
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
										aria-label={`${m.as_itself()}: ${permission.name}`}
										checked={clientIds.includes(permission.id)}
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

		<!-- The permission list can outgrow the dialog, so the actions stick to its bottom edge and cover its padding, keeping the usual 20px above them -->
		<Dialog.Footer class="bg-popover sticky -bottom-5 -mx-6 -mt-2 -mb-5 px-6 pt-3 pb-5">
			<Button variant="ghost" onclick={() => (open = false)}>{m.cancel()}</Button>
			<Button isLoading={saving} onclick={save}>{m.save_access()}</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
