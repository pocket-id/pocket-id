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

// Lists are joined the way the current language joins them, e.g. "A, B and C" or "A、B和C"
export function formatList(items: string[]) {
	return new Intl.ListFormat(getLocale(), { type: 'conjunction' }).format(items);
}

// Long lists name the first few and let the last entry count the rest, so a note stays short
export function formatNames(names: string[], limit: number, rest: (count: number) => string) {
	if (names.length <= limit) return formatList(names);
	return formatList([...names.slice(0, limit - 1), rest(names.length - limit + 1)]);
}
