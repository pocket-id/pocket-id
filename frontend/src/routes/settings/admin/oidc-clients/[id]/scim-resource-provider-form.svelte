<script lang="ts">
	import { openConfirmDialog } from '$lib/components/confirm-dialog';
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import ScimService from '$lib/services/scim-service';
	import type { ScimServiceProvider, ScimServiceProviderCreate } from '$lib/types/scim.type';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { emptyToUndefined } from '$lib/utils/zod-util';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod/v4';

	let {
		onSave,
		existingProvider,
		oidcClientId
	}: {
		existingProvider?: ScimServiceProvider;
		onSave: (provider: ScimServiceProviderCreate | null) => Promise<boolean>;
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

	const { inputs, ...form } = createForm<FormSchema>(formSchema, serviceProvider);

	async function onSubmit() {
		const data = form.validate();
		if (!data) return false;
		return await onSave({
			...data,
			oidcClientId
		});
	}

	async function onDisable() {
		openConfirmDialog({
			title: m.disable_scim_provisioning(),
			message: m.disable_scim_provisioning_confirm_description({
				clientName: existingProvider!.oidcClient.name
			}),
			confirm: {
				label: m.disable(),
				destructive: true,
				action: async () => {
					await onSave(null);
					form.setValue('endpoint', '');
					form.setValue('token', '');
				}
			}
		});
	}

	async function onSync() {
		const hasChanges = Object.keys($inputs).some(
			// @ts-ignore
			(key) => $inputs[key].value !== (existingProvider as any)[key]
		);

		if (hasChanges) {
			openConfirmDialog({
				title: m.save_changes_question(),
				message: m.scim_save_changes_description(),
				confirm: {
					label: m.save_and_sync(),
					action: async () => {
						const saved = await onSubmit();
						if (saved) {
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

<form onsubmit={preventDefault(onSubmit)}>
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
			{/if}
			<Button type="submit">{existingProvider ? m.save() : m.enable()}</Button>
		</div>
	</div>
</form>
