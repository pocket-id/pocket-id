import { afterNavigate, goto } from '$app/navigation';
import unsavedChanges from '#lib/stores/unsaved-changes-store.svelte.ts';

export const backNavigate = (defaultRoute: string) => {
	let previousUrl: URL | undefined;
	afterNavigate((e) => {
		if (e.from && !e.shallow) {
			previousUrl = new URL(e.from.url.href);
		}
	});

	const go = () => {
		if (previousUrl && previousUrl.pathname === defaultRoute) {
			window.history.back();
		} else {
			goto(defaultRoute);
		}
	};

	return {
		go,
		// Used once the entity is deleted, so pending edits are dropped instead of blocking the navigation
		leave: () => {
			unsavedChanges.discardAll();
			go();
		}
	};
};
