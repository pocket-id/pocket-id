import { afterNavigate, goto } from '$app/navigation';

export const backNavigate = (defaultRoute: string) => {
	let previousUrl: URL | undefined;
	afterNavigate((e) => {
		console.log(e);
		if (e.from) {
			previousUrl = e.from.url;
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
