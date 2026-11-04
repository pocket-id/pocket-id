<script lang="ts">
	import { buttonVariants } from '@pocket-id/ui/button';
	import * as DropdownMenu from '@pocket-id/ui/dropdown-menu';
	import { m } from '#lib/paraglide/messages.js';
	import userStore from '#lib/stores/user-store.ts';
	import type { AccessibleOidcClient } from '#lib/types/oidc.type.ts';
	import { encodeClientIdParam } from '#lib/utils/client-id-util.ts';
	import { cn } from '@pocket-id/ui/utils';
	import ArrowUpRightIcon from '~icons/solar/arrow-right-up-linear';
	import BanIcon from '~icons/solar/forbidden-circle-bold';
	import EllipsisIcon from '~icons/solar/menu-dots-bold';
	import LinkIcon from '~icons/solar/link-bold';
	import PencilIcon from '~icons/solar/pen-bold';
	import PinIcon from '~icons/solar/pin-bold';
	import { toast } from 'svelte-sonner';

	let {
		client,
		pinned,
		onTogglePin,
		onRevoke,
		class: className
	}: {
		client: AccessibleOidcClient;
		pinned?: boolean;
		// Only apps with a launch link can be pinned, so the others leave it out
		onTogglePin?: () => void;
		onRevoke: (client: AccessibleOidcClient) => void;
		class?: string;
	} = $props();

	// Pinning moves the app to the other list, and a leaving element freezes its menu mid-close, so the move waits until the menu has closed
	let pinRequested = false;

	function onOpenChangeComplete(open: boolean) {
		if (open || !pinRequested) return;
		pinRequested = false;
		onTogglePin?.();
	}

	async function copyLink() {
		await navigator.clipboard.writeText(client.launchURL!);
		toast.success(m.link_to_name_copied({ name: client.name }));
	}
</script>

{#if client.launchURL || $userStore?.isAdmin || client.lastUsedAt}
	<DropdownMenu.Root {onOpenChangeComplete}>
		<DropdownMenu.Trigger
			class={cn(
				buttonVariants({ variant: 'ghost', size: 'icon-sm' }),
				'relative z-10 shrink-0',
				className
			)}
			aria-label={m.actions_for_name({ name: client.name })}
		>
			<EllipsisIcon />
		</DropdownMenu.Trigger>
		<DropdownMenu.Content align="end">
			{#if onTogglePin}
				<DropdownMenu.Item onclick={() => (pinRequested = true)}>
					<PinIcon />
					{pinned ? m.unpin() : m.pin()}
				</DropdownMenu.Item>
			{/if}
			{#if client.launchURL}
				<DropdownMenu.Item>
					{#snippet child({ props })}
						<a {...props} href={client.launchURL} target="_blank" rel="noopener noreferrer">
							<ArrowUpRightIcon />
							{m.open_in_new_tab()}
						</a>
					{/snippet}
				</DropdownMenu.Item>
				<DropdownMenu.Item onclick={copyLink}>
					<LinkIcon />
					{m.copy_link()}
				</DropdownMenu.Item>
			{/if}
			{#if $userStore?.isAdmin}
				<DropdownMenu.Item>
					{#snippet child({ props })}
						<a {...props} href="/settings/admin/oidc-clients/{encodeClientIdParam(client.id)}">
							<PencilIcon />
							{m.edit_client()}
						</a>
					{/snippet}
				</DropdownMenu.Item>
			{/if}
			{#if client.lastUsedAt}
				{#if client.launchURL || $userStore?.isAdmin}
					<DropdownMenu.Separator />
				{/if}
				<DropdownMenu.Item variant="destructive" onclick={() => onRevoke(client)}>
					<BanIcon />
					{m.revoke_access_ellipsis()}
				</DropdownMenu.Item>
			{/if}
		</DropdownMenu.Content>
	</DropdownMenu.Root>
{/if}
