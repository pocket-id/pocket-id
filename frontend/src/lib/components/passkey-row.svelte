<script lang="ts">
	import ActionsMenu from '#lib/components/actions-menu.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { getLocale } from '#lib/paraglide/runtime.js';
	import type { Passkey } from '#lib/types/passkey.type.ts';
	import { authenticatorIconUrl } from '#lib/utils/cached-image-util.ts';
	import { LucideKeyRound, LucidePencil, LucideTrash2 } from '@lucide/svelte';
	import { mode } from 'mode-watcher';

	let {
		passkey,
		onRename,
		onDelete
	}: {
		passkey: Passkey;
		onRename?: () => void;
		onDelete: () => void;
	} = $props();

	const iconUrl = $derived(
		passkey.hasIcon ? authenticatorIconUrl(passkey.aaguid, mode.current === 'light') : undefined
	);

	// Falls back to the key glyph when the authenticator icon cannot be loaded
	let iconFailed = $state(false);

	// The row is reused across passkeys and themes, so a previous failure must not hide an icon that is now a different URL
	$effect(() => {
		void iconUrl;
		iconFailed = false;
	});
</script>

<div class="flex min-h-(--row-2line) items-center gap-3 px-4 py-2.5">
	{#if iconUrl && !iconFailed}
		<span
			class="bg-surface-raised inset-ring-border grid size-8 shrink-0 place-items-center rounded-[8px] inset-ring"
		>
			<img src={iconUrl} alt="" class="size-5 object-contain" onerror={() => (iconFailed = true)} />
		</span>
	{:else}
		<span class="bg-fill-muted text-fg-muted grid size-8 shrink-0 place-items-center rounded-[8px]">
			<LucideKeyRound class="size-4" />
		</span>
	{/if}
	<div class="flex min-w-0 flex-1 flex-col gap-0.5">
		<p class="text-heading truncate">{passkey.name}</p>
		<p class="text-caption text-fg-muted">
			{m.added_on()}
			{new Date(passkey.createdAt).toLocaleDateString(getLocale(), { dateStyle: 'medium' })}
		</p>
	</div>
	{#if onRename}
		<ActionsMenu
			item={passkey}
			size="icon-sm"
			label={m.actions_for_name({ name: passkey.name })}
			actions={[
				{ label: m.rename(), icon: LucidePencil, onClick: onRename },
				{ label: m.delete_ellipsis(), icon: LucideTrash2, variant: 'danger', onClick: onDelete }
			]}
		/>
	{:else}
		<Button
			variant="ghost"
			size="icon-xs"
			aria-label={m.delete_name_action({ name: passkey.name })}
			onclick={onDelete}
		>
			<LucideTrash2 />
		</Button>
	{/if}
</div>
