import { m } from '#lib/paraglide/messages.js';
import type { Component } from 'svelte';
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
		// Destructive confirms lead with this icon (a trash can unless given) so they never depend on color alone
		icon?: Component;
		action: () => unknown;
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
		icon?: Component;
		action: () => unknown;
	};
}) {
	// Starts from the defaults rather than the previous dialog, so a label, icon or destructive flag never carries over
	confirmDialogStore.set({
		open: true,
		title,
		message,
		confirm: {
			label: m.confirm(),
			destructive: false,
			...confirm
		}
	});
}

export { ConfirmDialog, openConfirmDialog };
