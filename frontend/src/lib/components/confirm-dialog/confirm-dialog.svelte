<script lang="ts">
	import * as AlertDialog from '@pocket-id/ui/alert-dialog';
	import { m } from '#lib/paraglide/messages.js';
	import Trash2Icon from '~icons/solar/trash-bin-minimalistic-bold';
	import { confirmDialogStore } from '.';
	import FormattedMessage from '../formatted-message.svelte';
	import { Button } from '@pocket-id/ui/button';

	let busy = $state(false);
	let cancelButton = $state<HTMLButtonElement | null>(null);

	const confirm = $derived($confirmDialogStore.confirm);
	const Icon = $derived(confirm.icon ?? Trash2Icon);

	// The dialog stays open while an async action runs, so the confirm button can show its progress
	async function onConfirm() {
		busy = true;
		try {
			await confirm.action();
		} finally {
			busy = false;
			$confirmDialogStore.open = false;
		}
	}
</script>

<AlertDialog.Root
	bind:open={() => $confirmDialogStore.open, (open) => !busy && ($confirmDialogStore.open = open)}
>
	<AlertDialog.Content
		class="z-9999"
		onOpenAutoFocus={(e) => {
			// Cancel takes focus, so Enter never confirms unless the confirm button was focused on purpose
			e.preventDefault();
			cancelButton?.focus();
		}}
	>
		<AlertDialog.Header>
			<AlertDialog.Title>{$confirmDialogStore.title}</AlertDialog.Title>
			<AlertDialog.Description>
				{#if typeof $confirmDialogStore.message === 'string'}
					{$confirmDialogStore.message}
				{:else}
					<FormattedMessage {...$confirmDialogStore.message} />
				{/if}
			</AlertDialog.Description>
		</AlertDialog.Header>
		<AlertDialog.Footer>
			<AlertDialog.Cancel bind:ref={cancelButton} disabled={busy}>{m.cancel()}</AlertDialog.Cancel>
			<!-- The label repeats the name, so it truncates and leaves room for Cancel while the title shows the name in full -->
			<Button
				variant={confirm.destructive ? 'destructive' : 'default'}
				class="min-w-0 shrink"
				isLoading={busy}
				onclick={onConfirm}
			>
				{#if confirm.destructive}<Icon class="size-3.5" />{/if}
				<span class="truncate">{confirm.label}</span>
			</Button>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>
