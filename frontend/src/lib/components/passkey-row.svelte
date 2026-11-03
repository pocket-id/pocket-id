<script lang="ts">
	import ActionsMenu from '#lib/components/actions-menu.svelte';
	import { Button } from '@pocket-id/ui/button';
	import { m } from '#lib/paraglide/messages.js';
	import { getLocale } from '#lib/paraglide/runtime.js';
	import type { Passkey } from '#lib/types/passkey.type.ts';
	import { authenticatorIconUrl } from '#lib/utils/cached-image-util.ts';
	import KeyRoundIcon from '~icons/solar/key-minimalistic-bold-duotone';
	import PencilIcon from '~icons/solar/pen-bold';
	import Trash2Icon from '~icons/solar/trash-bin-minimalistic-bold';
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
			<KeyRoundIcon class="size-4" />
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
				{ label: m.rename(), icon: PencilIcon, onClick: onRename },
				{ label: m.delete_ellipsis(), icon: Trash2Icon, variant: 'danger', onClick: onDelete }
			]}
		/>
	{:else}
		<Button
			variant="ghost"
			size="icon-xs"
			aria-label={m.delete_name_action({ name: passkey.name })}
			onclick={onDelete}
		>
			<Trash2Icon />
		</Button>
	{/if}
</div>
