<script lang="ts">
	import * as AlertDialog from '$lib/components/ui/alert-dialog';
	import { confirmDialogStore } from '.';
	import Button from '../ui/button/button.svelte';
</script>

<AlertDialog.Root bind:open={$confirmDialogStore.open}>
	<AlertDialog.Content>
		<AlertDialog.Header>
			<AlertDialog.Title>{$confirmDialogStore.title}</AlertDialog.Title>
			<AlertDialog.Description>
				{$confirmDialogStore.message}
			</AlertDialog.Description>
		</AlertDialog.Header>
		<AlertDialog.Footer>
			<AlertDialog.Cancel>Cancel</AlertDialog.Cancel>
			<AlertDialog.Action asChild>
				<Button
					variant={$confirmDialogStore.confirm.destructive ? 'destructive' : 'default'}
					onclick={() => {
						$confirmDialogStore.confirm.action();
						$confirmDialogStore.open = false;
					}}
				>
					{$confirmDialogStore.confirm.label}
				</Button>
			</AlertDialog.Action>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>
