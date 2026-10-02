import AppConfigService from '#lib/services/app-config-service.ts';
import type { AppConfig } from '#lib/types/application-configuration.type.ts';
import { applyAccentColor } from '#lib/utils/accent-color-util.ts';
import { writable } from 'svelte/store';

const appConfigStore = writable<AppConfig>();

const appConfigService = new AppConfigService();

const reload = async () => {
	const appConfig = await appConfigService.list();
	set(appConfig);
};

const set = (appConfig: AppConfig) => {
	applyAccentColor(appConfig.accentColor);
	appConfigStore.set(appConfig);
};

export default {
	subscribe: appConfigStore.subscribe,
	reload,
	set
};
