<script lang="ts">
	import * as AlertDialog from '$lib/components/ui/alert-dialog';
	import { confirmDialogStore } from '.';
	import FormattedMessage from '../formatted-message.svelte';
	import Button from '../ui/button/button.svelte';
</script>

<AlertDialog.Root bind:open={$confirmDialogStore.open}>
	<AlertDialog.Content class="z-9999">
		<AlertDialog.Header>
			<AlertDialog.Title>{$confirmDialogStore.title}</AlertDialog.Title>
			<AlertDialog.Description>
				<FormattedMessage m={$confirmDialogStore.message} />
			</AlertDialog.Description>
		</AlertDialog.Header>
		<AlertDialog.Footer>
			<AlertDialog.Cancel>Cancel</AlertDialog.Cancel>
			<AlertDialog.Action>
				{#snippet child()}
					<Button
						variant={$confirmDialogStore.confirm.destructive ? 'destructive' : 'default'}
						onclick={() => {
							$confirmDialogStore.confirm.action();
							$confirmDialogStore.open = false;
						}}
					>
						{$confirmDialogStore.confirm.label}
					</Button>
				{/snippet}
			</AlertDialog.Action>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>
