import {
	extractLocaleFromCookie,
	setLocale as setParaglideLocale,
	type Locale
} from '$lib/paraglide/runtime';
import { setDefaultOptions } from 'date-fns';
import { z } from 'zod/v4';

export async function setLocale(locale: Locale, reload = true) {
	await setLocaleForLibraries(locale);
	setParaglideLocale(locale, { reload });
}

export async function setLocaleForLibraries(
	locale: Locale = (extractLocaleFromCookie() as Locale) || 'en'
) {
	let dateFnsLocale: string = locale;
	if (dateFnsLocale === 'en') {
		dateFnsLocale = 'en-US'; // datefns doesn't have 'en'
	}

	const [zodResult, dateFnsResult] = await Promise.allSettled([
		import(`../../../node_modules/zod/v4/locales/${locale}.js`),
		import(`../../../node_modules/date-fns/locale/${dateFnsLocale}.js`)
	]);

	if (zodResult.status === 'fulfilled') {
		z.config(zodResult.value.default());
	} else {
		console.warn(`Failed to load zod locale for ${locale}:`, zodResult.reason);
	}

	if (dateFnsResult.status === 'fulfilled') {
		setDefaultOptions({
			locale: dateFnsResult.value.default
		});
	} else {
		console.warn(`Failed to load date-fns locale for ${locale}:`, dateFnsResult.reason);
	}
}
