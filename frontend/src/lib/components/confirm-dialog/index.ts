import { m } from '$lib/paraglide/messages';
import { writable } from 'svelte/store';
import ConfirmDialog from './confirm-dialog.svelte';

export const confirmDialogStore = writable({
	open: false,
	title: '',
	message: '',
	confirm: {
		label: m.confirm() as string,
		destructive: false,
		action: () => {}
	}
});

function openConfirmDialog({
	title,
	message,
	confirm
}: {
	title: string;
	message: string;
	confirm: {
		label?: string;
		destructive?: boolean;
		action: () => void;
	};
}) {
	confirmDialogStore.update((val) => ({
		open: true,
		title,
		message,
		confirm: {
			...val.confirm,
			...confirm
		}
	}));
}

export { ConfirmDialog, openConfirmDialog };
