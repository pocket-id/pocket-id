import AppConfigService from '$lib/services/app-config-service';
import type { AppConfig } from '$lib/types/application-configuration';
import { writable } from 'svelte/store';
import { applyAccentColor } from '$lib/utils/accent-color-util';

const appConfigStore = writable<AppConfig>();

const appConfigService = new AppConfigService();

const reload = async () => {
	const appConfig = await appConfigService.list();
	if (appConfig.accentColor) {
		applyAccentColor(appConfig.accentColor);
	}
	appConfigStore.set(appConfig);
};

const set = (appConfig: AppConfig) => {
	if (appConfig.accentColor) {
		applyAccentColor(appConfig.accentColor);
	}
	appConfigStore.set(appConfig);
};

export default {
	subscribe: appConfigStore.subscribe,
	reload,
	set
};
