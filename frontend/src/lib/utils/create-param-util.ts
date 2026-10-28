import { afterNavigate, replaceState } from '$app/navigation';

// Links such as the command palette and the My apps empty state open a list's create sheet with ?new
// The parameter is dropped right away, so a reload or Back does not open the sheet again
export function openOnCreateParam(open: () => void) {
	afterNavigate(({ to }) => {
		if (!to?.url.searchParams.has('new')) return;
		const url = new URL(to.url);
		url.searchParams.delete('new');
		replaceState(url, {});
		open();
	});
}
