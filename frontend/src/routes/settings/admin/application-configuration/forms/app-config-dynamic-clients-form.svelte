<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import UrlListInput from '$lib/components/form/url-list-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Field from '$lib/components/ui/field';
	import { m } from '$lib/paraglide/messages';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { AllAppConfig } from '$lib/types/application-configuration.type';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod/v4';

	let {
		appConfig,
		callback
	}: {
		appConfig: AllAppConfig;
		callback: (updatedConfig: Partial<AllAppConfig>) => Promise<void>;
	} = $props();

	let cimdUrlAllowlist: string[] = $derived(appConfig.cimdUrlAllowlist || []);
	let isLoading = $state(false);

	const formSchema = z.object({
		dynamicClientRetentionDays: z.number().int().min(0).max(36500)
	});
	let { inputs, ...form } = $derived(
		createForm(formSchema, {
			dynamicClientRetentionDays: appConfig.dynamicClientRetentionDays
		})
	);

	async function onSubmit() {
		const data = form.validate();
		if (!data) return;
		isLoading = true;

		const update: Partial<AllAppConfig> = {
			dynamicClientRetentionDays: data.dynamicClientRetentionDays
		};
		if ($appConfigStore.cimdEnabled) {
			update.cimdUrlAllowlist = cimdUrlAllowlist.filter((u) => u.trim() !== '');
		}

		await callback(update).finally(() => (isLoading = false));
		toast.success(m.application_configuration_updated_successfully());
	}
</script>

<form onsubmit={preventDefault(onSubmit)}>
	<fieldset class="flex flex-col gap-5" disabled={$appConfigStore.uiConfigDisabled}>
		<FormInput
			label={m.dynamic_client_retention()}
			type="number"
			description={m.dynamic_client_retention_description()}
			bind:input={$inputs.dynamicClientRetentionDays}
		/>

		{#if $appConfigStore.cimdEnabled}
			<Field.Field>
				<Field.Label>{m.cimd_url_allowlist()}</Field.Label>
				<Field.Description>
					{m.cimd_url_allowlist_description()}
				</Field.Description>
				<UrlListInput bind:urls={cimdUrlAllowlist} testIdPrefix="cimd-url-allowlist" />
			</Field.Field>
		{/if}

		<div class="flex justify-end pt-2">
			<Button {isLoading} type="submit">{m.save()}</Button>
		</div>
	</fieldset>
</form>
