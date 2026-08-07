<script lang="ts">
	import FormattedMessage from '$lib/components/formatted-message.svelte';
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

	let dynamicClientRedirectUriAllowlist: string[] = $derived(
		appConfig.dynamicClientRedirectUriAllowlist || []
	);
	let isLoading = $state(false);

	async function onSubmit() {
		isLoading = true;

		const update: Partial<AllAppConfig> = {
			dynamicClientRedirectUriAllowlist: dynamicClientRedirectUriAllowlist.filter(
				(u) => u.trim() !== ''
			)
		};

		await callback(update).finally(() => (isLoading = false));
		toast.success(m.application_configuration_updated_successfully());
	}
</script>

{#snippet dynamicClientRedirectUriAllowlistDescription()}
	<FormattedMessage message={m.dynamic_client_redirect_uri_allowlist_description} />
{/snippet}

<form onsubmit={preventDefault(onSubmit)}>
	<fieldset class="flex flex-col gap-5" disabled={$appConfigStore.uiConfigDisabled}>
		<FormInput
			label={m.dynamic_client_redirect_uri_allowlist()}
			description={dynamicClientRedirectUriAllowlistDescription}
		>
			<UrlListInput
				bind:urls={dynamicClientRedirectUriAllowlist}
				testIdPrefix="dynamic-client-redirect-uri-allowlist"
			/>
		</FormInput>

		<div class="flex justify-end pt-2">
			<Button {isLoading} type="submit">{m.save()}</Button>
		</div>
	</fieldset>
</form>
