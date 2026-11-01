<script lang="ts">
	import OidcClientAvatar from '#lib/components/oidc-client-avatar.svelte';
	import { buttonVariants } from '#lib/components/ui/button/index.ts';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import userStore from '#lib/stores/user-store.ts';
	import type { AccessibleOidcClient } from '#lib/types/oidc.type.ts';
	import { encodeClientIdParam } from '#lib/utils/client-id-util.ts';
	import { cn } from '#lib/utils/style.ts';
	import {
		LucideArrowUpRight,
		LucideBan,
		LucideEllipsis,
		LucideLink,
		LucidePencil
	} from '@lucide/svelte';
	import { formatDistanceToNow } from 'date-fns';
	import { toast } from 'svelte-sonner';

	let {
		client,
		row = false,
		onRevoke
	}: {
		client: AccessibleOidcClient;
		// Apps without a launch link are always rows; the others are tiles from the sm breakpoint up
		row?: boolean;
		onRevoke: (client: AccessibleOidcClient) => void;
	} = $props();

	const host = $derived(client.launchURL && new URL(client.launchURL).hostname);
	const tile = $derived(!row);

	async function copyLink() {
		await navigator.clipboard.writeText(client.launchURL!);
		toast.success(m.link_to_name_copied({ name: client.name }));
	}
</script>

<li
	class={cn(
		'group/app relative isolate flex min-h-(--row-2line) items-center gap-3 py-2 shadow-[inset_0_-1px_0_var(--border)]',
		tile &&
			'sm:bg-surface sm:hover:bg-fill-row-hover sm:min-h-[140px] sm:flex-col sm:items-stretch sm:gap-0 sm:rounded-md sm:p-4 sm:shadow-[inset_0_0_0_1px_var(--border)] sm:transition-[background-color,box-shadow,scale] sm:duration-(--dur-fast) sm:ease-out sm:hover:shadow-[inset_0_0_0_1px_var(--border-strong)] sm:has-[a:active]:scale-(--press-scale)'
	)}
>
	<span
		class={cn(
			'flex',
			tile && 'max-sm:*:size-6 max-sm:*:rounded-[6px] max-sm:*:text-[11px] sm:mb-3'
		)}
	>
		<OidcClientAvatar
			id={client.id}
			name={client.name}
			hasLogo={client.hasLogo}
			size={row ? 24 : 40}
		/>
	</span>

	<div
		class={cn(
			'flex min-w-0 flex-1 flex-col',
			row ? 'sm:flex-row sm:items-baseline sm:justify-between sm:gap-4' : 'sm:gap-0.5'
		)}
	>
		<div
			class={cn(
				'grid min-w-0 grid-cols-[auto_minmax(0,1fr)] items-baseline gap-2',
				tile && 'sm:flex sm:flex-col sm:items-stretch sm:gap-0.5'
			)}
		>
			{#if client.launchURL}
				<!-- The name link stretches over the whole tile so the tile is one hit target, and the menu button sits above it -->
				<a
					href={client.launchURL}
					target="_blank"
					rel="noopener noreferrer"
					class="text-control text-fg focus-visible:after:outline-ring truncate font-medium outline-none after:absolute after:inset-0 focus-visible:after:outline-2 focus-visible:after:-outline-offset-2 sm:after:rounded-md"
				>
					{client.name}
				</a>
			{:else}
				<span class="text-control text-fg truncate font-medium">{client.name}</span>
			{/if}
			{#if host}
				<span class="text-caption text-fg-muted truncate">{host}</span>
			{/if}
		</div>
		<span class="text-caption text-fg-muted truncate">
			{client.lastUsedAt
				? m.app_used_ago({ time: formatDistanceToNow(client.lastUsedAt, { addSuffix: true }) })
				: m.not_signed_in_yet()}
		</span>
	</div>

	{#if client.launchURL}
		<LucideArrowUpRight
			aria-hidden="true"
			class={cn('text-fg-muted size-3.5 shrink-0', tile && 'sm:hidden')}
		/>
	{/if}

	{#if client.launchURL || $userStore?.isAdmin || client.lastUsedAt}
		<DropdownMenu.Root>
			<DropdownMenu.Trigger
				class={cn(
					buttonVariants({ variant: 'ghost', size: 'icon-sm' }),
					'relative z-10 shrink-0',
					tile && 'sm:absolute sm:top-3 sm:right-3'
				)}
				aria-label={m.actions_for_name({ name: client.name })}
			>
				<LucideEllipsis />
			</DropdownMenu.Trigger>
			<DropdownMenu.Content align="end">
				{#if client.launchURL}
					<DropdownMenu.Item>
						{#snippet child({ props })}
							<a {...props} href={client.launchURL} target="_blank" rel="noopener noreferrer">
								<LucideArrowUpRight />
								{m.open_in_new_tab()}
							</a>
						{/snippet}
					</DropdownMenu.Item>
					<DropdownMenu.Item onclick={copyLink}>
						<LucideLink />
						{m.copy_link()}
					</DropdownMenu.Item>
				{/if}
				{#if $userStore?.isAdmin}
					<DropdownMenu.Item>
						{#snippet child({ props })}
							<a {...props} href="/settings/admin/oidc-clients/{encodeClientIdParam(client.id)}">
								<LucidePencil />
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
						<LucideBan />
						{m.revoke_access_ellipsis()}
					</DropdownMenu.Item>
				{/if}
			</DropdownMenu.Content>
		</DropdownMenu.Root>
	{/if}
</li>
