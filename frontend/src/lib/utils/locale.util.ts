import { setLocale as setParaglideLocale, type Locale } from '$lib/paraglide/runtime';
import { z } from 'zod/v4';

export function setLocale(locale: Locale, reload = true) {
	import(`../../../node_modules/zod/dist/esm/v4/locales/${locale}.js`)
		.then((zodLocale) => z.config(zodLocale.default()))
		.finally(() => {
			setParaglideLocale(locale, { reload });
		});
}
