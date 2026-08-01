<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import UrlListInput from '$lib/components/form/url-list-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { AllAppConfig } from '$lib/types/application-configuration.type';
	import { preventDefault } from '$lib/utils/event-util';
	import { toast } from 'svelte-sonner';

	let {
		appConfig,
		callback
	}: {
		appConfig: AllAppConfig;
		callback: (updatedConfig: Partial<AllAppConfig>) => Promise<void>;
	} = $props();

	let cimdUrlAllowlist: string[] = $derived(appConfig.cimdUrlAllowlist || []);
	let isLoading = $state(false);

	async function onSubmit() {
		isLoading = true;

		const update: Partial<AllAppConfig> = {
			cimdUrlAllowlist: cimdUrlAllowlist.filter((u) => u.trim() !== '')
		};

		await callback(update).finally(() => (isLoading = false));
		toast.success(m.application_configuration_updated_successfully());
	}
</script>

<form onsubmit={preventDefault(onSubmit)}>
	<fieldset class="flex flex-col gap-5" disabled={$appConfigStore.uiConfigDisabled}>
		<FormInput label={m.cimd_url_allowlist()} description={m.cimd_url_allowlist_description()}>
			<UrlListInput bind:urls={cimdUrlAllowlist} testIdPrefix="cimd-url-allowlist" />
		</FormInput>

		<div class="flex justify-end pt-2">
			<Button {isLoading} type="submit">{m.save()}</Button>
		</div>
	</fieldset>
</form>
