<script lang="ts">
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
	import FormInput from '#lib/components/form/form-input.svelte';
	import SettingRow from '#lib/components/setting-row.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import { Badge } from '#lib/components/ui/badge/index.ts';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Field from '#lib/components/ui/field/index.ts';
	import { Input } from '#lib/components/ui/input/index.ts';
	import { Switch } from '#lib/components/ui/switch/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import AppConfigService from '#lib/services/app-config-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { AllAppConfig } from '#lib/types/application-configuration.type.ts';
	import { axiosErrorToast, getAxiosErrorMessage } from '#lib/utils/error-util.ts';
	import { createForm, pickSchemaValues } from '#lib/utils/form-util.ts';
	import { trackFormChanges } from '#lib/utils/unsaved-changes-util.svelte.ts';
	import {
		LucideBan,
		LucideImage,
		LucideRefreshCw,
		LucideShieldOff,
		LucideUserX
	} from '@lucide/svelte';
	import { tick } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod/v4';

	let {
		callback,
		appConfig
	}: {
		appConfig: AllAppConfig;
		callback: (appConfig: Partial<AllAppConfig>) => Promise<void>;
	} = $props();

	const appConfigService = new AppConfigService();
	const appName = $derived($appConfigStore.appName);

	let ldapEnabled = $state(appConfig.ldapEnabled);
	let syncing = $state(false);
	let syncError = $state<string | null>(null);

	const formSchema = z.object({
		ldapUrl: z.url(),
		ldapBindDn: z.string().min(1),
		ldapBindPassword: z.string().min(1),
		ldapBase: z.string().min(1),
		ldapUserSearchFilter: z.string().min(1),
		ldapUserGroupSearchFilter: z.string().min(1),
		ldapSkipCertVerify: z.boolean(),
		ldapAttributeUserUniqueIdentifier: z.string().min(1),
		ldapAttributeUserUsername: z.string().min(1),
		ldapAttributeUserEmail: z.string().min(1),
		ldapAttributeUserFirstName: z.string().min(1),
		ldapAttributeUserLastName: z.string().optional(),
		ldapAttributeUserDisplayName: z.string().optional(),
		ldapAttributeUserProfilePicture: z.string().optional(),
		ldapAttributeGroupMember: z.string().optional(),
		ldapAttributeGroupUniqueIdentifier: z.string().min(1),
		ldapAttributeGroupName: z.string().min(1),
		ldapAdminGroupName: z.string().optional(),
		ldapSoftDeleteUsers: z.boolean(),
		ldapAllowProfilePictureOverride: z.boolean()
	});

	const formStore = createForm(formSchema, pickSchemaValues(formSchema, appConfig));
	const inputs = formStore.inputs;

	const mapping: {
		key: Exclude<keyof z.infer<typeof formSchema>, 'ldapSkipCertVerify' | 'ldapSoftDeleteUsers'>;
		label: string;
		placeholder: string;
		note?: string;
	}[] = [
		{
			key: 'ldapAttributeUserUniqueIdentifier',
			label: m.config_map_user_id(),
			placeholder: 'entryUUID',
			note: m.config_note_required_never_change()
		},
		{
			key: 'ldapAttributeUserUsername',
			label: m.username(),
			placeholder: 'uid',
			note: m.config_note_required()
		},
		{
			key: 'ldapAttributeUserEmail',
			label: m.email(),
			placeholder: 'mail',
			note: m.config_note_required()
		},
		{
			key: 'ldapAttributeUserFirstName',
			label: m.first_name(),
			placeholder: 'givenName',
			note: m.config_note_required()
		},
		{ key: 'ldapAttributeUserLastName', label: m.last_name(), placeholder: 'sn' },
		{ key: 'ldapAttributeUserDisplayName', label: m.display_name(), placeholder: 'cn' },
		{
			key: 'ldapAttributeUserProfilePicture',
			label: m.profile_picture(),
			placeholder: 'jpegPhoto',
			note: m.config_note_picture()
		},
		{ key: 'ldapAttributeGroupMember', label: m.config_map_group_members(), placeholder: 'member' },
		{
			key: 'ldapAttributeGroupUniqueIdentifier',
			label: m.config_map_group_id(),
			placeholder: 'entryUUID',
			note: m.config_note_required_never_change()
		},
		{
			key: 'ldapAttributeGroupName',
			label: m.config_map_group_name(),
			placeholder: 'cn',
			note: m.config_note_group_name()
		},
		{
			key: 'ldapAdminGroupName',
			label: m.config_map_admin_group(),
			placeholder: 'pocketid_admins',
			note: m.config_note_admin_group()
		}
	];

	async function saveLdap(data: z.infer<typeof formSchema>, enable: boolean) {
		await callback({ ...data, ldapEnabled: enable });
	}

	trackFormChanges(
		() => formStore,
		(data) => saveLdap(data, ldapEnabled)
	);

	async function syncLdap() {
		syncing = true;
		syncError = null;
		try {
			await appConfigService.syncLdap();
			return true;
		} catch (e) {
			syncError = getAxiosErrorMessage(e, m.an_unknown_error_occurred());
			return false;
		} finally {
			syncing = false;
		}
	}

	async function onSync() {
		if (await syncLdap()) toast.success(m.ldap_sync_finished());
	}

	async function onEnable() {
		const data = formStore.validate();
		if (!data) {
			await tick();
			document.querySelector<HTMLElement>('[aria-invalid="true"]')?.focus();
			return;
		}
		openConfirmDialog({
			title: m.config_enable_ldap_title({ host: new URL(data.ldapUrl).hostname }),
			message: m.config_enable_ldap_message({ appName }),
			confirm: {
				label: m.config_enable_ldap_confirm(),
				action: async () => {
					try {
						await saveLdap(data, true);
					} catch (e) {
						axiosErrorToast(e);
						return;
					}
					formStore.commit(data);
					ldapEnabled = true;
					if (await syncLdap()) toast.success(m.config_ldap_enabled_and_synced());
				}
			}
		});
	}

	function onDisable() {
		openConfirmDialog({
			title: m.config_disable_ldap_title(),
			message: m.config_disable_ldap_message({ appName }),
			confirm: {
				label: m.config_disable_ldap_confirm(),
				destructive: true,
				icon: LucideBan,
				action: async () => {
					await callback({ ldapEnabled: false });
					ldapEnabled = false;
					syncError = null;
					toast.success(m.config_ldap_disabled());
				}
			}
		});
	}
</script>

<div class="flex min-w-0 flex-col gap-10">
	<SettingsSection title={m.status()}>
		<div class="flex min-h-13 flex-wrap items-center gap-x-4 gap-y-3 py-3">
			<div class="flex min-w-0 flex-1 basis-64 flex-col gap-0.5" aria-live="polite">
				{#if syncing}
					<Badge variant="muted">{m.config_ldap_syncing()}</Badge>
				{:else if syncError}
					<Badge variant="destructive">{m.config_ldap_sync_failed()}</Badge>
					<span class="text-fg-muted text-mono-s ps-3.5 font-mono wrap-anywhere">{syncError}</span>
				{:else if ldapEnabled}
					<Badge variant="success">{m.config_ldap_connected()}</Badge>
				{:else}
					<Badge variant="muted">{m.config_ldap_not_connected()}</Badge>
				{/if}
			</div>
			<div class="flex flex-wrap items-center gap-2">
				{#if ldapEnabled}
					<Button size="sm" variant="secondary" isLoading={syncing} onclick={onSync}>
						<LucideRefreshCw />
						{m.sync_now()}
					</Button>
					<Button
						size="sm"
						variant="ghost"
						disabled={$appConfigStore.uiConfigDisabled || syncing}
						onclick={onDisable}
					>
						{m.config_disable_ldap()}
					</Button>
				{:else}
					<Button
						size="sm"
						variant="secondary"
						disabled={$appConfigStore.uiConfigDisabled}
						onclick={onEnable}
					>
						{m.config_enable_ldap()}
					</Button>
				{/if}
			</div>
		</div>
	</SettingsSection>

	<fieldset class="contents" disabled={$appConfigStore.uiConfigDisabled}>
		<SettingsSection
			title={m.config_connection()}
			description={m.config_connection_description({ appName })}
		>
			<div class="flex flex-col gap-4 py-4 shadow-[inset_0_-1px_0_var(--border)]">
				<FormInput
					class="max-w-[400px]"
					inputClass="font-mono"
					label={m.config_server_url()}
					placeholder="ldaps://ldap.example.com:636"
					bind:input={$inputs.ldapUrl}
				/>
				<FormInput
					inputClass="font-mono"
					label={m.config_bind_dn()}
					placeholder="cn=pocketid,ou=services,dc=example,dc=com"
					bind:input={$inputs.ldapBindDn}
				/>
				<FormInput
					class="max-w-[400px]"
					inputClass="font-mono"
					label={m.config_bind_password()}
					type="password"
					bind:input={$inputs.ldapBindPassword}
				/>
				<FormInput
					inputClass="font-mono"
					label={m.config_base_dn()}
					placeholder="dc=example,dc=com"
					bind:input={$inputs.ldapBase}
				/>
			</div>
			<SettingRow
				title={m.skip_certificate_verification()}
				description={m.config_self_signed_only()}
				icon={LucideShieldOff}
			>
				{#snippet control(props)}
					<Switch {...props} bind:checked={$inputs.ldapSkipCertVerify.value} />
				{/snippet}
			</SettingRow>
		</SettingsSection>

		<SettingsSection
			title={m.config_search()}
			description={m.config_search_description({ appName })}
		>
			<div class="flex flex-col gap-4 py-4 shadow-[inset_0_-1px_0_var(--border)]">
				<FormInput
					inputClass="font-mono"
					label={m.config_user_search_filter()}
					placeholder="(objectClass=person)"
					bind:input={$inputs.ldapUserSearchFilter}
				/>
				<FormInput
					inputClass="font-mono"
					label={m.config_group_search_filter()}
					placeholder="(objectClass=groupOfNames)"
					bind:input={$inputs.ldapUserGroupSearchFilter}
				/>
			</div>
			<SettingRow
				title={m.config_keep_removed_ldap_users()}
				description={m.config_keep_removed_ldap_users_description()}
				icon={LucideUserX}
			>
				{#snippet control(props)}
					<Switch {...props} bind:checked={$inputs.ldapSoftDeleteUsers.value} />
				{/snippet}
			</SettingRow>
			<SettingRow
				title={m.ldap_allow_profile_picture_override()}
				description={m.ldap_allow_profile_picture_override_description()}
				icon={LucideImage}
			>
				{#snippet control(props)}
					<Switch {...props} bind:checked={$inputs.ldapAllowProfilePictureOverride.value} />
				{/snippet}
			</SettingRow>
		</SettingsSection>

		<SettingsSection title={m.attribute_mapping()}>
			<!-- On phones each mapping becomes a block of label, input and note inside one box -->
			<table
				class="text-control mt-2 w-full table-fixed max-sm:block max-sm:rounded-md max-sm:shadow-[inset_0_0_0_1px_var(--border)]"
			>
				<thead class="text-caption text-fg-muted text-start max-sm:sr-only">
					<tr>
						<th scope="col" class="w-40 py-2 pe-3 text-start font-semibold">
							{m.config_pocket_id_field()}
						</th>
						<th scope="col" class="w-56 py-2 pe-3 text-start font-semibold">
							{m.config_ldap_attribute()}
						</th>
						<th scope="col" class="py-2 text-start font-semibold">{m.config_notes()}</th>
					</tr>
				</thead>
				<tbody class="max-sm:block">
					{#each mapping as row (row.key)}
						{@const input = $inputs[row.key]!}
						<tr
							class="shadow-[inset_0_-1px_0_var(--border)] last:shadow-none max-sm:flex max-sm:flex-col max-sm:gap-1.5 max-sm:p-3"
						>
							<th
								scope="row"
								class="py-1.5 pe-3 text-start font-normal max-sm:p-0 max-sm:font-medium"
							>
								<label for="ldap-map-{row.key}">{row.label}</label>
							</th>
							<td class="py-1.5 pe-3 max-sm:p-0">
								<Input
									id="ldap-map-{row.key}"
									class="h-(--h-sm) font-mono"
									placeholder={row.placeholder}
									aria-invalid={!!input.error}
									aria-describedby={input.error || row.note
										? `ldap-map-${row.key}-note`
										: undefined}
									bind:value={
										() => input.value as string | undefined,
										(value) => {
											input.value = value;
											input.error = null;
										}
									}
								/>
							</td>
							<td class="text-caption py-1.5 max-sm:p-0 max-sm:empty:hidden">
								{#if input.error}
									<Field.Error id="ldap-map-{row.key}-note">{input.error}</Field.Error>
								{:else if row.note}
									<span id="ldap-map-{row.key}-note" class="text-fg-muted">{row.note}</span>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</SettingsSection>
	</fieldset>
</div>
