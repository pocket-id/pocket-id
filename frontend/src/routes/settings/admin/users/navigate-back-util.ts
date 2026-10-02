import { afterNavigate, goto } from '$app/navigation';

export const backNavigate = (defaultRoute: string) => {
	let previousUrl: URL | undefined;
	afterNavigate((e) => {
		if (e.from && !e.shallow) {
			previousUrl = new URL(e.from.url.href);
		}
	});

	return {
		go: () => {
			if (previousUrl && previousUrl.pathname === defaultRoute) {
				window.history.back();
			} else {
				goto(defaultRoute);
			}
		}
	};
};
