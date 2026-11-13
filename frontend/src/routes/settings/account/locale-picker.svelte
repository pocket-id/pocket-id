<script lang="ts">
	import SearchableSelect from '#lib/components/form/searchable-select.svelte';
	import { getLocale, type Locale } from '#lib/paraglide/runtime.js';
	import UserService from '#lib/services/user-service.ts';
	import { setLocale } from '#lib/utils/locale.util.ts';

	let controlProps: { id: string; 'aria-describedby': string | undefined } = $props();

	const userService = new UserService();
	const currentLocale = getLocale();

	const locales = {
		ca: 'Català',
		cs: 'Čeština',
		da: 'Dansk',
		de: 'Deutsch',
		en: 'English',
		es: 'Español',
		et: 'Eesti',
		fi: 'Suomi',
		fr: 'Français',
		hu: 'Magyar',
		it: 'Italiano',
		ja: '日本語',
		ko: '한국어',
		lv: 'Latviešu',
		nl: 'Nederlands',
		no: 'Norsk',
		pl: 'Polski',
		pt: 'Português',
		'pt-BR': 'Português brasileiro',
		ru: 'Русский',
		sv: 'Svenska',
		tr: 'Türkçe',
		uk: 'Українська',
		vi: 'Tiếng Việt',
		'zh-CN': '简体中文',
		'zh-TW': '繁體中文（臺灣）'
	};

	async function updateLocale(locale: Locale) {
		await userService.updateCurrent({ locale });
		await setLocale(locale);
	}
</script>

<SearchableSelect
	{...controlProps}
	class="h-(--h-sm) w-fit min-w-40 pr-2 pl-3"
	items={Object.entries(locales).map(([value, label]) => ({ value, label }))}
	value={currentLocale}
	onSelect={(locale) => updateLocale(locale as Locale)}
/>
