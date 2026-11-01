<script lang="ts">
	import { page } from '$app/state';
	import TokenPreviewInputs from '#lib/components/token-preview-inputs.svelte';
	import * as Tabs from '#lib/components/ui/tabs/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import OidcClaimMappingPolicyService from '#lib/services/oidc-claim-mapping-policy-service.ts';
	import type {
		OidcClaimMappingDto,
		OidcClaimMappingsPreview
	} from '#lib/types/oidc-claim-mapping-policy.type.ts';
	import { debounced } from '#lib/utils/debounce-util.ts';
	import { getAxiosErrorMessage } from '#lib/utils/error-util.ts';
	import { cn } from '#lib/utils/style.ts';

	type Token = keyof OidcClaimMappingsPreview;

	let {
		mappings,
		highlighted
	}: {
		mappings: OidcClaimMappingDto[];
		// The claim whose row is hovered or focused, whose lines light up
		highlighted: string | null;
	} = $props();

	const policyService = new OidcClaimMappingPolicyService();

	let userId = $state('');
	let scopes = $state(['openid', 'email', 'profile', 'groups']);
	let token = $state<Token>('idToken');
	let preview = $state<OidcClaimMappingsPreview | null>(null);
	let errorMessage = $state<string | null>(null);

	// Pocket ID sets these itself, so they lead the token and are dimmed to set them apart from what the policy releases
	const ownClaims = $derived.by((): Record<string, unknown> => {
		if (token === 'userInfo') return { sub: userId };
		const iat = Math.floor(Date.now() / 1000);
		return { iss: page.url.origin, sub: userId, exp: iat + 3600, iat };
	});

	// The ID token and userinfo are null when openid isn't requested, because a real authorization wouldn't issue them
	const issued = $derived(!preview || preview[token] !== null);

	const lines = $derived.by(() => {
		const mapped = Object.entries(preview?.[token] ?? {}).filter(([key]) => !(key in ownClaims));
		return [
			...Object.entries(ownClaims).map(([key, value]) => ({ key, value, claim: null })),
			// Keys no mapping names come from the custom claim wildcard
			...mapped.map(([key, value]) => ({
				key,
				value,
				claim: mappings.some((m) => m.claimName === key) ? key : '*'
			}))
		];
	});

	const skipped = $derived(
		mappings.filter(
			(mapping) =>
				mapping.claimName !== '*' &&
				!(mapping[token] && mapping.scope.some((s) => scopes.includes(s)))
		).length
	);

	// Each request gets a number so that responses arriving out of order don't overwrite newer ones
	let latestRequest = 0;
	const refresh = debounced(
		async (userId: string, scopes: string[], mappings: OidcClaimMappingDto[]) => {
			const request = ++latestRequest;
			try {
				const data = await policyService.previewClaimMappings(userId, scopes, mappings);
				if (request !== latestRequest) return;
				preview = data;
				errorMessage = null;
			} catch (e) {
				if (request === latestRequest) errorMessage = getAxiosErrorMessage(e);
			}
		},
		200
	);

	$effect(() => {
		if (userId) refresh(userId, $state.snapshot(scopes), $state.snapshot(mappings));
	});
</script>

{#snippet jsonValue(value: unknown)}
	{#if value === null}<span class="text-fg-muted">null</span
		>{:else if Array.isArray(value)}[{#each value as item, i (i)}{#if i > 0},
			{/if}{@render jsonValue(item)}{/each}]{:else if typeof value === 'string'}<span
			class="text-primary">{JSON.stringify(value)}</span
		>{:else if typeof value === 'object'}{JSON.stringify(value)}{:else}<span class="text-fg-muted"
			>{String(value)}</span
		>{/if}
{/snippet}

<aside
	aria-labelledby="claims-preview-title"
	class="bg-surface-sunken flex min-w-0 flex-col gap-3 rounded-lg p-4 shadow-[inset_0_0_0_1px_var(--border)] xl:sticky xl:top-6"
>
	<h3 id="claims-preview-title" class="text-heading text-fg">{m.preview()}</h3>

	<TokenPreviewInputs bind:userId bind:scopes />

	<Tabs.Root bind:value={token}>
		<Tabs.List aria-label={m.token()} class="w-full">
			<Tabs.Trigger value="idToken" class="flex-1">{m.id_token()}</Tabs.Trigger>
			<Tabs.Trigger value="accessToken" class="flex-1">{m.access_token()}</Tabs.Trigger>
			<Tabs.Trigger value="userInfo" class="flex-1">{m.userinfo()}</Tabs.Trigger>
		</Tabs.List>
	</Tabs.Root>

	<div aria-live="polite">
		{#if errorMessage}
			<div
				class="bg-surface flex flex-col gap-1 rounded-md px-4 py-6 text-center shadow-[inset_0_0_0_1px_var(--border)]"
			>
				<p class="text-heading">{m.couldnt_load_the_preview()}</p>
				<p class="text-control text-fg-muted">{errorMessage}</p>
			</div>
		{:else if !issued}
			<p
				class="bg-surface text-control text-fg-muted rounded-md px-4 py-6 text-center shadow-[inset_0_0_0_1px_var(--border)]"
			>
				{m.only_issued_with_openid_scope()}
			</p>
		{:else}
			<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
			<pre
				tabindex="0"
				aria-label={m.token_as_json()}
				class={cn(
					'bg-surface text-mono-s focus-visible:outline-ring m-0 rounded-md py-3 font-mono leading-5 wrap-anywhere whitespace-pre-wrap shadow-[inset_0_0_0_1px_var(--border)] transition-opacity duration-(--dur-base) focus-visible:outline-2 focus-visible:outline-offset-2',
					!preview && 'opacity-50'
				)}><span class="block px-3">{'{'}</span>{#each lines as line, i (line.key)}<span
						class={cn(
							'block ps-7 pe-3 transition-colors duration-(--dur-fast)',
							!line.claim && 'text-fg-subtle [&_span]:text-inherit',
							line.claim && line.claim === highlighted && 'bg-fill-selected'
						)}
						><span class={line.claim ? 'text-fg' : ''}>{JSON.stringify(line.key)}</span
						>: {@render jsonValue(line.value)}{i < lines.length - 1 ? ',' : ''}</span
					>{/each}<span class="block px-3">{'}'}</span></pre>
		{/if}
	</div>

	{#if issued && skipped}
		<p class="text-caption text-fg-muted">
			{skipped === 1 ? m.claims_not_released_one() : m.claims_not_released({ count: skipped })}
		</p>
	{/if}
</aside>
