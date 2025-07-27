<script lang="ts">
	import * as Select from '$lib/components/ui/select';
	import { getLocale, type Locale } from '$lib/paraglide/runtime';
	import UserService from '$lib/services/user-service';
	import userStore from '$lib/stores/user-store';
	import { setLocale } from '$lib/utils/locale.util';

	const userService = new UserService();
	const currentLocale = getLocale();

	const locales = {
		cs: 'Čeština',
		da: 'Dansk',
		de: 'Deutsch',
		en: 'English',
		es: 'Español',
		fr: 'Français',
		it: 'Italiano',
		nl: 'Nederlands',
		pl: 'Polski',
		'pt-BR': 'Português brasileiro',
		ru: 'Русский',
		uk: 'Українська',
		vi: 'Tiếng Việt',
		'zh-CN': '简体中文',
		'zh-TW': '繁體中文（臺灣）'
	};

	async function updateLocale(locale: Locale) {
		await userService.updateCurrent({
			...$userStore!,
			locale
		});
		setLocale(locale);
	}
</script>

<Select.Root type="single" value={currentLocale} onValueChange={(v) => updateLocale(v as Locale)}>
	<Select.Trigger class="h-9 max-w-[200px]" aria-label="Select locale">
		{locales[currentLocale]}
	</Select.Trigger>
	<Select.Content>
		{#each Object.entries(locales) as [value, label]}
			<Select.Item {value}>{label}</Select.Item>
		{/each}
	</Select.Content>
</Select.Root>
