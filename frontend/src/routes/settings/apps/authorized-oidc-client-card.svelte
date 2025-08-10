<script lang="ts">
	import { goto } from '$app/navigation';
	import ImageBox from '$lib/components/image-box.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import { m } from '$lib/paraglide/messages';
	import userStore from '$lib/stores/user-store';
	import type { AuthorizedOidcClient, OidcClientMetaData } from '$lib/types/oidc.type';
	import { cachedApplicationLogo, cachedOidcClientLogo } from '$lib/utils/cached-image-util';
	import {
		LucideBan,
		LucideEllipsisVertical,
		LucideExternalLink,
		LucidePencil
	} from '@lucide/svelte';
	import { mode } from 'mode-watcher';

	let {
		authorizedClient,
		onRevoke
	}: {
		authorizedClient: AuthorizedOidcClient;
		onRevoke: (client: OidcClientMetaData) => Promise<void>;
	} = $props();

	const isLightMode = $derived(mode.current === 'light');
</script>

<Card.Root
	class="border-muted group h-[140px] p-5 transition-all duration-200 hover:shadow-md"
	data-testid="authorized-oidc-client-card"
>
	<Card.Content class=" p-0">
		<div class="flex gap-3">
			<div class="aspect-square h-[56px]">
				<ImageBox
					class="grow rounded-lg object-contain"
					src={authorizedClient.client.hasLogo
						? cachedOidcClientLogo.getUrl(authorizedClient.client.id)
						: cachedApplicationLogo.getUrl(isLightMode)}
					alt={m.name_logo({ name: authorizedClient.client.name })}
				/>
			</div>
			<div class="flex w-full justify-between gap-3">
				<div>
					<div class="mb-1 flex items-start gap-2">
						<h3
							class="text-foreground line-clamp-2 leading-tight font-semibold break-words break-all text-ellipsis"
						>
							{authorizedClient.client.name}
						</h3>
					</div>
					{#if authorizedClient.client.launchURL}
						<p
							class="text-muted-foreground line-clamp-1 text-xs break-words break-all text-ellipsis"
						>
							{new URL(authorizedClient.client.launchURL).hostname}
						</p>
					{/if}
				</div>
				<div>
					<DropdownMenu.Root>
						<DropdownMenu.Trigger>
							<LucideEllipsisVertical class="size-4" />
							<span class="sr-only">{m.toggle_menu()}</span>
						</DropdownMenu.Trigger>
						<DropdownMenu.Content align="end">
							<DropdownMenu.Item
								onclick={() => goto(`/settings/admin/oidc-clients/${authorizedClient.client.id}`)}
								><LucidePencil class="mr-2 size-4" /> {m.edit()}</DropdownMenu.Item
							>
							{#if $userStore?.isAdmin}
								<DropdownMenu.Item
									class="text-red-500 focus:!text-red-700"
									onclick={() => onRevoke(authorizedClient.client)}
									><LucideBan class="mr-2 size-4" />{m.revoke()}</DropdownMenu.Item
								>
							{/if}
						</DropdownMenu.Content>
					</DropdownMenu.Root>
				</div>
			</div>
		</div>

		<div class="mt-2 flex justify-end">
			<Button
				href={authorizedClient.client.launchURL}
				target="_blank"
				size="sm"
				class="h-8 text-xs"
				disabled={!authorizedClient.client.launchURL}
			>
				{m.launch()}
				<LucideExternalLink class="ml-1 size-3" />
			</Button>
		</div>
	</Card.Content>
</Card.Root>

<style>
</style>
