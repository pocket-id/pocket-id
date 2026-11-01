import { getLocale, setLocale as setParaglideLocale, type Locale } from '#lib/paraglide/runtime.js';
import { setDefaultOptions } from 'date-fns';
import { z } from 'zod/v4';
import { validationErrorMap } from './validation-error-util';

export async function setLocale(locale: Locale, reload = true) {
	await setLocaleForLibraries(locale);
	setParaglideLocale(locale, { reload });
}

export async function setLocaleForLibraries(locale: Locale = getLocale()) {
	document.documentElement.lang = locale;

	let dateFnsLocale: string = locale;
	if (dateFnsLocale === 'en') {
		dateFnsLocale = 'en-US'; // datefns doesn't have 'en'
	}

	z.config({ customError: validationErrorMap });

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
