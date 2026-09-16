<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import UrlListInput from '$lib/components/form/url-list-input.svelte';
	import FormattedMessage from '$lib/components/formatted-message.svelte';
	import { m } from '$lib/paraglide/messages';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { AllAppConfig } from '$lib/types/application-configuration.type';
	import { trackUnsavedValue } from '$lib/utils/unsaved-changes-util.svelte';

	let {
		appConfig,
		callback
	}: {
		appConfig: AllAppConfig;
		callback: (updatedConfig: Partial<AllAppConfig>) => Promise<void>;
	} = $props();

	let cimdUrlAllowlist: string[] = $state(appConfig.cimdUrlAllowlist ?? []);
	if (cimdUrlAllowlist.length === 0) cimdUrlAllowlist.push('');

	trackUnsavedValue(
		() => cimdUrlAllowlist.filter((u) => u.trim() !== ''),
		(value) => {
			cimdUrlAllowlist = value;
			if (cimdUrlAllowlist.length === 0) cimdUrlAllowlist.push('');
		},
		(value) => callback({ cimdUrlAllowlist: value })
	);
</script>

{#snippet cimdUrlAllowlistDescription()}
	<FormattedMessage message={m.cimd_url_allowlist_description} />
{/snippet}

<fieldset class="flex flex-col gap-5" disabled={$appConfigStore.uiConfigDisabled}>
	<FormInput label={m.cimd_url_allowlist()} description={cimdUrlAllowlistDescription}>
		<UrlListInput bind:urls={cimdUrlAllowlist} testIdPrefix="cimd-url-allowlist" keepAtLeastOne />
	</FormInput>
</fieldset>
