<script lang="ts">
	import ActionsMenu from '#lib/components/actions-menu.svelte';
	import { Button } from '@pocket-id/ui/button';
	import * as Tooltip from '@pocket-id/ui/tooltip';
	import { m } from '#lib/paraglide/messages.js';
	import { getLocale } from '#lib/paraglide/runtime.js';
	import type { Passkey } from '#lib/types/passkey.type.ts';
	import { authenticatorIconUrl } from '#lib/utils/cached-asset-util.ts';
	import KeyRoundIcon from '~icons/solar/key-minimalistic-bold-duotone';
	import PencilIcon from '~icons/solar/pen-bold';
	import Trash2Icon from '~icons/solar/trash-bin-minimalistic-bold';
	import VerifiedIcon from '~icons/solar/verified-check-bold-duotone';
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

	const verifiedLabel = $derived(
		passkey.verifiedAuthenticatorName
			? m.verified_as_authenticator({ name: passkey.verifiedAuthenticatorName })
			: m.verified_authenticator()
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
			<KeyRoundIcon class="size-[18px]" />
		</span>
	{/if}
	<div class="flex min-w-0 flex-1 flex-col gap-0.5">
		<p class="text-heading wrap-anywhere">
			{passkey.name}
			{#if passkey.attestationVerified}
				<Tooltip.Root>
					<Tooltip.Trigger>
						{#snippet child({ props })}
							<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
							<span
								{...props}
								role="img"
								tabindex="0"
								aria-label={verifiedLabel}
								class="text-success-solid focus-visible:outline-ring inline-grid size-4 place-items-center rounded-sm align-[-2px] focus-visible:outline-2"
							>
								<VerifiedIcon class="size-4" />
							</span>
						{/snippet}
					</Tooltip.Trigger>
					<!-- Model names are long, so the label only wraps where the viewport ends -->
					<Tooltip.Content class="max-w-(--bits-tooltip-content-available-width)">
						{verifiedLabel}
					</Tooltip.Content>
				</Tooltip.Root>
			{/if}
		</p>
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
