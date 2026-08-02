import { m } from '$lib/paraglide/messages';
import { writable } from 'svelte/store';
import type { AnyFormattedMessage, AnyMessage, FormattedMessageProps } from '../formatted-message';
import ConfirmDialog from './confirm-dialog.svelte';

interface ConfirmDialogState {
	open: boolean;
	title: string;
	message: string | AnyFormattedMessage;
	confirm: {
		label: string;
		destructive: boolean;
		action: () => void;
	};
}

export const confirmDialogStore = writable<ConfirmDialogState>({
	open: false,
	title: '',
	message: '',
	confirm: {
		label: m.confirm() as string,
		destructive: false,
		action: () => {}
	}
});

function openConfirmDialog<TMessage extends AnyMessage = AnyMessage>({
	title,
	message,
	confirm
}: {
	title: string;
	message: string | FormattedMessageProps<TMessage>;
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
