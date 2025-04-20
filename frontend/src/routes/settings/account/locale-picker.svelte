<script lang="ts">
	import * as Select from '$lib/components/ui/select';
	import { getLocale, setLocale, type Locale } from '$lib/paraglide/runtime';
	import UserService from '$lib/services/user-service';
	import userStore from '$lib/stores/user-store';

	const userService = new UserService();
	const currentLocale = getLocale();

	const locales = {
		'cs-CZ': 'Čeština',
		'de-DE': 'Deutsch',
		'en-US': 'English',
		'fr-FR': 'Français',
		'nl-NL': 'Nederlands',
		'pt-BR': 'Português brasileiro',
		'ru-RU': 'Русский',
		'it-IT': 'Italiano'
	};

	async function updateLocale(locale: Locale) {
		await userService.updateCurrent({
			...$userStore!,
			locale
		});
		setLocale(locale);
	}
</script>

<Select.Root
	selected={{
		label: locales[currentLocale],
		value: currentLocale
	}}
	onSelectedChange={(v) => updateLocale(v!.value)}
>
	<Select.Trigger class="h-9 max-w-[200px]" aria-label="Select locale">
		<Select.Value>{locales[currentLocale]}</Select.Value>
	</Select.Trigger>
	<Select.Content>
		{#each Object.entries(locales) as [value, label]}
			<Select.Item {value}>{label}</Select.Item>
		{/each}
	</Select.Content>
</Select.Root>
