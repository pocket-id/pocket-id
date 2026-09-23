<script lang="ts">
	import SwitchWithLabel from '$lib/components/form/switch-with-label.svelte';
	import { m } from '$lib/paraglide/messages';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { AllAppConfig } from '$lib/types/application-configuration.type';
	import { createForm } from '$lib/utils/form-util';
	import { trackFormChanges } from '$lib/utils/unsaved-changes-util.svelte';
	import { z } from 'zod/v4';

	let {
		appConfig,
		callback
	}: {
		appConfig: AllAppConfig;
		callback: (appConfig: Partial<AllAppConfig>) => Promise<void>;
	} = $props();

	const formSchema = z.object({ autoCreateOidcClientSecret: z.boolean() });
	let formStore = $derived(
		createForm(formSchema, { autoCreateOidcClientSecret: appConfig.autoCreateOidcClientSecret })
	);
	let inputs = $derived(formStore.inputs);

	trackFormChanges(() => formStore, callback);
</script>

<fieldset disabled={$appConfigStore.uiConfigDisabled}>
	<SwitchWithLabel
		id="auto-create-oidc-client-secret"
		label={m.auto_create_client_secret()}
		description={m.auto_create_client_secret_description()}
		bind:checked={$inputs.autoCreateOidcClientSecret.value}
	/>
</fieldset>
