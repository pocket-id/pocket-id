<script lang="ts">
	import { openConfirmDialog } from '$lib/components/confirm-dialog';
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import ScimService from '$lib/services/scim-service';
	import type { ScimServiceProvider, ScimServiceProviderCreate } from '$lib/types/scim.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { trackFormChanges } from '$lib/utils/unsaved-changes-util.svelte';
	import { emptyToUndefined } from '$lib/utils/zod-util';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod/v4';

	let {
		onSave,
		existingProvider,
		oidcClientId
	}: {
		existingProvider?: ScimServiceProvider;
		onSave: (provider: ScimServiceProviderCreate | null) => Promise<void>;
		oidcClientId: string;
	} = $props();

	const scimService = new ScimService();

	let isSyncing = $state(false);

	const serviceProvider = {
		endpoint: existingProvider?.endpoint || '',
		token: existingProvider?.token || ''
	};

	const formSchema = z.object({
		endpoint: z.url(),
		token: emptyToUndefined(z.string())
	});
	type FormSchema = typeof formSchema;

	const formStore = createForm<FormSchema>(formSchema, serviceProvider);
	const { inputs } = formStore;

	async function saveProvider(data: z.infer<FormSchema>) {
		await onSave({
			...data,
			oidcClientId
		});
	}

	// Enable/disable/sync have their own buttons rather than going through the unsaved-changes
	// bar, so they report their outcome themselves.
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
		if (await trySaveProvider()) toast.success(m.scim_enabled_successfully());
	}

	// Tracked by the unsaved-changes bar once the provider exists; until then the Enable button
	// saves it.
	trackFormChanges(() => formStore, saveProvider, { enabled: () => !!existingProvider });

	async function onDisable() {
		openConfirmDialog({
			title: m.disable_scim_provisioning(),
			message: {
				message: m.disable_scim_provisioning_confirm_description,
				inputs: { clientName: existingProvider!.oidcClient.name }
			},
			confirm: {
				label: m.disable(),
				destructive: true,
				action: async () => {
					try {
						await onSave(null);
						toast.success(m.scim_disabled_successfully());
					} catch (e) {
						axiosErrorToast(e);
						return;
					}
					formStore.setValue('endpoint', '');
					formStore.setValue('token', '');
					// The cleared fields are the saved state now, so they aren't unsaved changes.
					formStore.commit({ endpoint: '', token: undefined });
				}
			}
		});
	}

	async function onSync() {
		const hasChanges =
			$inputs.endpoint.value !== existingProvider?.endpoint ||
			($inputs.token?.value ?? '') !== (existingProvider?.token ?? '');

		if (hasChanges) {
			openConfirmDialog({
				title: m.save_changes_question(),
				message: m.scim_save_changes_description(),
				confirm: {
					label: m.save_and_sync(),
					action: async () => {
						if (await trySaveProvider()) {
							syncProvider();
						}
					}
				}
			});
		} else {
			syncProvider();
		}
	}

	async function syncProvider() {
		isSyncing = true;
		await scimService
			.syncServiceProvider(existingProvider!.id)
			.then(() => {
				existingProvider = {
					...existingProvider!,
					lastSyncedAt: new Date().toISOString()
				};
				toast.success(m.scim_sync_successful());
			})
			.catch(() => toast.error(m.scim_sync_failed()))
			.finally(() => (isSyncing = false));
	}
</script>

<form onsubmit={preventDefault(onEnable)}>
	<div class="flex flex-col gap-3 sm:flex-row">
		<div class="w-full">
			<FormInput
				placeholder="https://scim.example.com/v2"
				label={m.scim_endpoint()}
				bind:input={$inputs.endpoint}
			/>
		</div>
		<div class="w-full">
			<FormInput label={m.scim_token()} bind:input={$inputs.token} type="password" />
		</div>
	</div>
	<div
		class="mt-5 flex items-end flex-col sm:flex-row {existingProvider
			? 'justify-between'
			: 'justify-end'} "
	>
		{#if existingProvider}
			<p class="text-muted-foreground text-xs self-start sm:self-auto">
				{m.last_successful_sync_at({
					time: existingProvider.lastSyncedAt
						? new Date(existingProvider.lastSyncedAt).toLocaleString()
						: m.never()
				})}
			</p>
		{/if}
		<div class="mt-5 flex justify-end gap-3">
			{#if existingProvider}
				<Button variant="destructive" onclick={onDisable}>{m.disable()}</Button>
				<Button variant="secondary" isLoading={isSyncing} onclick={onSync}>{m.sync_now()}</Button>
			{:else}
				<Button type="submit">{m.enable()}</Button>
			{/if}
		</div>
	</div>
</form>
