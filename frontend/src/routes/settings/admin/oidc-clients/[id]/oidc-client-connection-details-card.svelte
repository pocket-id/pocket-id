<script lang="ts">
	import CopyToClipboard from '#lib/components/copy-to-clipboard.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Card from '#lib/components/ui/card/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import clientSecretStore, { autoCreatedSecretId } from '#lib/stores/client-secret-store.ts';
	import type {
		OidcClient,
		OidcClientSecret,
		OidcDiscoveryConfiguration
	} from '#lib/types/oidc.type.ts';
	import { cn } from '#lib/utils/style.ts';
	import { LucideChevronDown } from '@lucide/svelte';
	import { slide } from 'svelte/transition';

	let {
		client,
		secrets,
		oidcConfiguration
	}: {
		client: OidcClient;
		secrets: OidcClientSecret[];
		oidcConfiguration: OidcDiscoveryConfiguration;
	} = $props();

	let showAllEndpoints = $state(false);

	// The secret that was generated together with the client is only known until the page is left
	const createdSecret = $derived(
		$autoCreatedSecretId && secrets.some((secret) => secret.id === $autoCreatedSecretId)
			? $clientSecretStore[$autoCreatedSecretId]
			: undefined
	);

	const endpoints = $derived([
		{ label: m.issuer_url(), value: oidcConfiguration.issuer },
		{ label: m.authorization_url(), value: oidcConfiguration.authorization_endpoint },
		{ label: m.token_url(), value: oidcConfiguration.token_endpoint },
		{ label: m.userinfo_url(), value: oidcConfiguration.userinfo_endpoint },
		{ label: m.logout_url(), value: oidcConfiguration.end_session_endpoint },
		{ label: m.certificate_url(), value: oidcConfiguration.jwks_uri }
	]);
</script>

{#snippet detail(label: string, value: string, testId?: string)}
	<div class="flex min-w-0 flex-col gap-1">
		<span class="text-muted-foreground text-xs">{label}</span>
		<CopyToClipboard {value}>
			<span class="font-mono text-xs break-all" data-testid={testId}>{value}</span>
		</CopyToClipboard>
	</div>
{/snippet}

<!-- The main values sit next to each other in one row, so they don't leave the full-width card mostly empty -->
<!-- The endpoints use a grid instead, so that their columns line up across rows -->
<Card.Root>
	<Card.Header>
		<Card.Title class="min-w-0 truncate">{client.name}</Card.Title>
		<Card.Action>
			<Button
				variant="ghost"
				size="sm"
				class="text-muted-foreground"
				aria-label={showAllEndpoints ? m.show_less_details() : m.show_more_details()}
				onclick={() => (showAllEndpoints = !showAllEndpoints)}
			>
				<!-- Only the chevron is shown on small screens so the title doesn't have to wrap -->
				<span class="hidden sm:inline">
					{showAllEndpoints ? m.show_less_details() : m.show_more_details()}
				</span>
				<LucideChevronDown
					class={cn(
						'size-4 opacity-60 transition-transform duration-200 sm:ml-1.5',
						showAllEndpoints && 'rotate-180'
					)}
				/>
			</Button>
		</Card.Action>
	</Card.Header>
	<Card.Content class="flex flex-col gap-4">
		<div class="flex flex-wrap gap-x-12 gap-y-4">
			{@render detail(m.client_id(), client.id, 'client-id')}
			{#if createdSecret}
				{@render detail(m.client_secret(), createdSecret, 'created-client-secret')}
			{/if}
			{@render detail(
				m.oidc_discovery_url(),
				`${oidcConfiguration.issuer}/.well-known/openid-configuration`
			)}
		</div>
		{#if showAllEndpoints}
			<div
				class="grid gap-x-6 gap-y-4 border-t pt-4 md:grid-cols-2 xl:grid-cols-3"
				transition:slide={{ duration: 200 }}
			>
				{#each endpoints as endpoint (endpoint.label)}
					{@render detail(endpoint.label, endpoint.value)}
				{/each}
			</div>
		{/if}
	</Card.Content>
</Card.Root>
