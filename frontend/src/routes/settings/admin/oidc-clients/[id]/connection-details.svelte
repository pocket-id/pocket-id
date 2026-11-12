<script lang="ts">
	import CopyRows from '#lib/components/copy-rows.svelte';
	import { Button } from '@pocket-id/ui/button';
	import { cn } from '@pocket-id/ui/utils';
	import { m } from '#lib/paraglide/messages.js';
	import type { OidcDiscoveryConfiguration } from '#lib/types/oidc.type.ts';
	import ChevronDownIcon from '~icons/solar/alt-arrow-down-linear';

	let {
		clientId,
		configuration
	}: {
		clientId: string;
		configuration: OidcDiscoveryConfiguration;
	} = $props();

	// Most apps configure themselves from the discovery URL, so the single endpoints wait behind a toggle
	let showEndpoints = $state(false);

	const rows = $derived([
		{ label: m.client_id(), value: clientId, copyLabel: m.copy_client_id(), testId: 'client-id' },
		{
			label: m.discovery_url(),
			value: `${configuration.issuer}/.well-known/openid-configuration`,
			copyLabel: m.copy_discovery_url()
		},
		{ label: m.issuer(), value: configuration.issuer, copyLabel: m.copy_issuer_url() },
		...(showEndpoints
			? [
					{
						label: m.authorization(),
						value: configuration.authorization_endpoint,
						copyLabel: m.copy_authorization_url()
					},
					{ label: m.token(), value: configuration.token_endpoint, copyLabel: m.copy_token_url() },
					{
						label: m.userinfo(),
						value: configuration.userinfo_endpoint,
						copyLabel: m.copy_userinfo_url()
					},
					{
						label: m.end_session(),
						value: configuration.end_session_endpoint,
						copyLabel: m.copy_end_session_url()
					},
					{ label: m.jwks(), value: configuration.jwks_uri, copyLabel: m.copy_jwks_url() }
				]
			: [])
	]);
</script>

<div class="flex flex-col items-stretch py-2">
	<CopyRows {rows} />
	<Button
		variant="ghost"
		size="sm"
		class="-ms-3 self-start"
		aria-expanded={showEndpoints}
		onclick={() => (showEndpoints = !showEndpoints)}
	>
		{showEndpoints ? m.hide_endpoints() : m.show_all_endpoints()}
		<ChevronDownIcon class={cn('transition-transform', showEndpoints && 'rotate-180')} />
	</Button>
</div>
