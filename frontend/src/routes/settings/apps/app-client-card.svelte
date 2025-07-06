<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import ImageBox from '$lib/components/image-box.svelte';
	import { m } from '$lib/paraglide/messages';
	import type { AccessibleOidcClient } from '$lib/types/oidc.type';
	import { cachedOidcClientLogo } from '$lib/utils/cached-image-util';
	import { LucideExternalLink, Settings, CheckCircle } from '@lucide/svelte';
	import { cachedApplicationLogo } from '$lib/utils/cached-image-util';
	import { mode } from 'mode-watcher';
	import userStore from '$lib/stores/user-store';

	interface Props {
		app: AccessibleOidcClient;
	}

	let { app }: Props = $props();

	const isLightMode = $derived(mode.current === 'light');

	function getAppUrl(client: AccessibleOidcClient): string {
		// Try to get the main callback URL as the app URL
		return client.callbackURLs?.[0]?.replace(/\/callback.*$/, '') || '#';
	}
</script>

<Card.Root
	class="border-muted group h-40 overflow-hidden transition-all duration-200 hover:shadow-md"
>
	<Card.Content class="flex h-full">
		<div class="mr-3 flex-shrink-0">
			<ImageBox
				class="h-20 w-20 rounded-lg"
				src={app.hasLogo
					? cachedOidcClientLogo.getUrl(app.id)
					: cachedApplicationLogo.getUrl(isLightMode)}
				alt={m.name_logo({ name: app.name })}
			/>
		</div>

		<div class="flex min-w-0 flex-1 flex-col justify-between py-1">
			<div class="min-h-0 flex-1">
				<div class="mb-1 flex items-start gap-2">
					<h3 class="text-foreground line-clamp-2 text-base font-semibold leading-tight">
						{app.name}
					</h3>
					{#if app.isAuthorized}
						<CheckCircle class="mt-0.5 size-4 flex-shrink-0 text-green-500" />
					{/if}
				</div>
				<p class="text-muted-foreground line-clamp-1 text-xs">
					{app.callbackURLs?.[0] ? new URL(app.callbackURLs[0]).hostname : 'Application'}
				</p>
			</div>

			<div class="mt-2">
				{#if $userStore?.isAdmin}
					<div class="flex gap-1.5">
						<Button
							href="/settings/admin/oidc-clients/{app.id}"
							size="sm"
							variant="outline"
							class="h-8 flex-1 px-2 text-xs"
						>
							<Settings class="mr-1 size-3" />
							{m.edit()}
						</Button>
						<Button
							href={getAppUrl(app)}
							target="_blank"
							size="sm"
							variant="default"
							class="h-8 flex-1 px-2 text-xs"
						>
							{m.launch()}
							<LucideExternalLink class="ml-1 size-3" />
						</Button>
					</div>
				{:else}
					<Button
						href={getAppUrl(app)}
						target="_blank"
						size="sm"
						class="h-8 w-full text-xs"
						variant={app.isAuthorized ? 'default' : 'outline'}
					>
						{app.isAuthorized ? m.launch() : m.authorize()}
						<LucideExternalLink class="ml-1 size-3" />
					</Button>
				{/if}
			</div>
		</div>
	</Card.Content>
</Card.Root>

<style>
	.line-clamp-1 {
		display: -webkit-box;
		-webkit-line-clamp: 1;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	.line-clamp-2 {
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
</style>
