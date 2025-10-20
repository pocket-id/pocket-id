<script lang="ts">
	import { goto } from '$app/navigation';
	import ImageBox from '$lib/components/image-box.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import { m } from '$lib/paraglide/messages';
	import userStore from '$lib/stores/user-store';
	import type { AccessibleOidcClient, OidcClientMetaData } from '$lib/types/oidc.type';
	import { cachedApplicationLogo, cachedOidcClientLogo } from '$lib/utils/cached-image-util';
	import {
		LucideBan,
		LucideEllipsisVertical,
		LucideExternalLink,
		LucideLogIn,
		LucidePencil
	} from '@lucide/svelte';
	import { formatDistanceToNow } from 'date-fns';
	import { mode } from 'mode-watcher';

	let {
		client,
		onRevoke
	}: {
		client: AccessibleOidcClient;
		onRevoke: (client: OidcClientMetaData) => Promise<void>;
	} = $props();

	const isLightMode = $derived(mode.current === 'light');
</script>

<Card.Root
	class="border-muted group relative h-[140px] p-5 transition-all duration-200 hover:shadow-md sm:max-w-[50vw] md:max-w-[400px]"
	data-testid="authorized-oidc-client-card"
>
	<Card.Content class=" p-0">
		<div class="flex gap-3">
			<div class="aspect-square h-[56px]">
				<ImageBox
					class="size-14"
					src={client.hasLogo
						? cachedOidcClientLogo.getUrl(client.id, isLightMode)
						: cachedApplicationLogo.getUrl(isLightMode)}
					alt={m.name_logo({ name: client.name })}
				/>
			</div>
			<div class="flex w-full justify-between gap-3">
				<div>
					<div class="mb-1 flex items-start gap-2">
						<h3
							class="text-foreground line-clamp-2 leading-tight font-semibold break-words break-all text-ellipsis"
						>
							{client.name}
						</h3>
					</div>
					{#if client.launchURL}
						<p
							class="text-muted-foreground line-clamp-1 text-xs break-words break-all text-ellipsis"
						>
							{new URL(client.launchURL).hostname}
						</p>
					{/if}
				</div>
				{#if $userStore?.isAdmin || client.lastUsedAt}
					<div>
						<DropdownMenu.Root>
							<DropdownMenu.Trigger>
								<LucideEllipsisVertical class="size-4" />
								<span class="sr-only">{m.toggle_menu()}</span>
							</DropdownMenu.Trigger>
							<DropdownMenu.Content align="end">
								{#if $userStore?.isAdmin}
									<DropdownMenu.Item
										onclick={() => goto(`/settings/admin/oidc-clients/${client.id}`)}
										><LucidePencil class="mr-2 size-4" /> {m.edit()}</DropdownMenu.Item
									>
								{/if}
								{#if client.lastUsedAt}
									<DropdownMenu.Item
										class="text-red-500 focus:!text-red-700"
										onclick={() => onRevoke(client)}
										><LucideBan class="mr-2 size-4" />{m.revoke()}</DropdownMenu.Item
									>
								{/if}
							</DropdownMenu.Content>
						</DropdownMenu.Root>
					</div>
				{/if}
			</div>
		</div>

		<div class="mt-2 flex items-end justify-between">
			{#if client.lastUsedAt}
				<Tooltip.Provider>
					<Tooltip.Root>
						<Tooltip.Trigger>
							<p class="text-muted-foreground flex items-center text-xs">
								<LucideLogIn class="mr-1 size-3" />
								{formatDistanceToNow(client.lastUsedAt, { addSuffix: true })}
							</p>
						</Tooltip.Trigger>
						<Tooltip.Content
							>{m.last_signed_in_ago({
								time: formatDistanceToNow(client.lastUsedAt)
							})}</Tooltip.Content
						>
					</Tooltip.Root></Tooltip.Provider
				>
			{:else}
				<div></div>
			{/if}
			<Button
				href={client.launchURL}
				target="_blank"
				size="sm"
				class="h-8 text-xs"
				rel="noopener noreferrer"
				disabled={!client.launchURL}
			>
				{m.launch()}
				<LucideExternalLink class="ml-1 size-3" />
			</Button>
		</div>
	</Card.Content>
</Card.Root>
