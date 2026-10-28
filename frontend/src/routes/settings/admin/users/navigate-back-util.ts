import { afterNavigate, goto } from '$app/navigation';
import unsavedChanges from '#lib/stores/unsaved-changes-store.svelte.ts';

// Leaves a deleted entity's page for its list, going back in history when the list was the previous page so its scroll and filters stay
export const backNavigate = (listRoute: string) => {
	let previousUrl: URL | undefined;
	afterNavigate((e) => {
		if (e.from && !e.shallow) {
			previousUrl = new URL(e.from.url.href);
		}
	});

	return {
		// Pending edits are dropped, so they don't block leaving a page whose entity is gone
		leave: () => {
			unsavedChanges.discardAll();
			if (previousUrl?.pathname === listRoute) {
				window.history.back();
			} else {
				goto(listRoute);
			}
		}
	};
};
