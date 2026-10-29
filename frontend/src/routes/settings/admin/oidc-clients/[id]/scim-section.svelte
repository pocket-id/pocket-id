<script lang="ts">
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
	import DocsLink from '#lib/components/docs-link.svelte';
	import FormInput from '#lib/components/form/form-input.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import { Badge } from '#lib/components/ui/badge/index.ts';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { getLocale } from '#lib/paraglide/runtime.js';
	import ScimService from '#lib/services/scim-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { OidcClient } from '#lib/types/oidc.type.ts';
	import type { ScimServiceProvider, ScimServiceProviderCreate } from '#lib/types/scim.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { preventDefault } from '#lib/utils/event-util.ts';
	import { createForm } from '#lib/utils/form-util.ts';
	import { trackFormChanges } from '#lib/utils/unsaved-changes-util.svelte.ts';
	import { emptyToUndefined } from '#lib/utils/zod-util.ts';
	import { LucideBan, LucidePlug, LucideRefreshCw } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod/v4';

	let {
		client,
		existingProvider,
		onSave
	}: {
		client: OidcClient;
		existingProvider?: ScimServiceProvider;
		onSave: (provider: ScimServiceProviderCreate | null) => Promise<void>;
	} = $props();

	const scimService = new ScimService();

	let isSyncing = $state(false);
	let lastSyncedAt = $state(existingProvider?.lastSyncedAt);

	const formSchema = z.object({
		endpoint: z.url(),
		token: emptyToUndefined(z.string())
	});
	type FormSchema = typeof formSchema;

	const formStore = createForm<FormSchema>(formSchema, {
		endpoint: existingProvider?.endpoint || '',
		token: existingProvider?.token || ''
	});
	const { inputs } = formStore;

	async function saveProvider(data: z.infer<FormSchema>) {
		await onSave({ ...data, oidcClientId: client.id });
	}

	// Enable, disable and sync have their own buttons rather than going through the unsaved-changes bar, so they report their outcome themselves
	async function trySaveProvider() {
		const data = formStore.validate();
		if (!data) return false;
		try {
			await saveProvider(data);
			formStore.commit(data);
			return true;
		} catch (e) {
			axiosErrorToast(e);
			return false;
		}
	}

	async function onEnable() {
		if (await trySaveProvider()) toast.success(m.provisioning_enabled());
	}

	// Tracked by the unsaved-changes bar once the provider exists; until then Enable provisioning saves it
	trackFormChanges(() => formStore, saveProvider, { enabled: () => !!existingProvider });

	function onDisable() {
		openConfirmDialog({
			title: m.disable_provisioning_question(),
			message: m.disable_provisioning_consequence({
				appName: $appConfigStore.appName,
				name: client.name
			}),
			confirm: {
				label: m.disable_provisioning(),
				destructive: true,
				icon: LucideBan,
				action: async () => {
					try {
						await onSave(null);
						toast.success(m.provisioning_disabled());
					} catch (e) {
						axiosErrorToast(e);
						return;
					}
					formStore.setValue('endpoint', '');
					formStore.setValue('token', '');
					// The cleared fields are the saved state now, so they aren't unsaved changes
					formStore.commit({ endpoint: '', token: undefined });
				}
			}
		});
	}

	function onSync() {
		if (!formStore.isDirty()) return syncProvider();
		openConfirmDialog({
			title: m.save_changes_question(),
			message: m.scim_save_changes_description(),
			confirm: {
				label: m.save_and_sync(),
				action: async () => {
					if (await trySaveProvider()) syncProvider();
				}
			}
		});
	}

	async function syncProvider() {
		isSyncing = true;
		try {
			await scimService.syncServiceProvider(existingProvider!.id);
			lastSyncedAt = new Date().toISOString();
			toast.success(m.name_synced({ name: client.name }));
		} catch {
			toast.error(m.scim_sync_failed());
		} finally {
			isSyncing = false;
		}
	}
</script>

<SettingsSection title={m.scim_provisioning()}>
	{#snippet description()}
		{m.scim_section_description({ appName: $appConfigStore.appName, name: client.name })}
		<DocsLink href="https://pocket-id.org/docs/configuration/scim" />
	{/snippet}
	<form onsubmit={preventDefault(onEnable)} class="flex flex-col gap-6 pt-2">
		{#if existingProvider}
			<div role="status">
				{#if lastSyncedAt}
					<Badge variant="success">
						{m.synced_date({
							date: new Date(lastSyncedAt).toLocaleString(getLocale(), {
								dateStyle: 'medium',
								timeStyle: 'short'
							})
						})}
					</Badge>
				{:else}
					<Badge variant="muted">{m.never_synced()}</Badge>
				{/if}
			</div>
		{/if}
		<FormInput
			label={m.scim_endpoint()}
			description={m.scim_endpoint_helper({ name: client.name })}
			placeholder="https://scim.example.com/v2"
			inputClass="font-mono"
			bind:input={$inputs.endpoint}
		/>
		<FormInput
			label={m.scim_token()}
			optional
			type="password"
			inputClass="font-mono"
			bind:input={$inputs.token}
		/>
		<div class="flex flex-wrap gap-2">
			{#if existingProvider}
				<Button variant="secondary" size="sm" isLoading={isSyncing} onclick={onSync}>
					<LucideRefreshCw />
					{m.sync_now()}
				</Button>
				<Button variant="ghost" size="sm" onclick={onDisable}>
					{m.disable_provisioning_ellipsis()}
				</Button>
			{:else}
				<Button type="submit" variant="secondary" size="sm">
					<LucidePlug />
					{m.enable_provisioning()}
				</Button>
			{/if}
		</div>
	</form>
</SettingsSection>
