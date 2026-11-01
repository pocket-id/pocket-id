import AppConfigService from '#lib/services/app-config-service.ts';
import type { AppConfig } from '#lib/types/application-configuration.type.ts';
import { applyColorTheme } from '#lib/utils/color-theme-util.ts';
import { writable } from 'svelte/store';

const appConfigStore = writable<AppConfig>();

const appConfigService = new AppConfigService();

const reload = async () => {
	const appConfig = await appConfigService.list();
	set(appConfig);
};

const set = (appConfig: AppConfig) => {
	applyColorTheme(appConfig.colorTheme);
	document.documentElement.classList.toggle('no-motion', appConfig.disableAnimations);
	appConfigStore.set(appConfig);
};

export default {
	subscribe: appConfigStore.subscribe,
	reload,
	set
};
