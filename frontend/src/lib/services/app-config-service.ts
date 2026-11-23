import userStore from '#lib/stores/user-store.ts';
import type {
	AllAppConfig,
	AppConfigRawResponse
} from '#lib/types/application-configuration.type.ts';
import {
	cachedApplicationLogo,
	cachedBackgroundImage,
	cachedDefaultProfilePicture,
	cachedEmailLogo,
	cachedProfilePicture,
	cachedTitleFont
} from '#lib/utils/cached-asset-util.ts';
import { get } from 'svelte/store';
import APIService from './api-service';

export default class AppConfigService extends APIService {
	list = async (showAll = false) => {
		const url = showAll ? '/admin/application-configuration' : '/application-configuration';
		const { data } = await this.api.get<AppConfigRawResponse>(url);
		return parseConfigList(data);
	};

	// The keys an environment variable sets are read-only in the UI
	listWithEnvOverrides = async () => {
		const { data } = await this.api.get<AppConfigRawResponse>('/admin/application-configuration');
		const envOverrides = new Set(
			data.filter((v) => v.envOverride).map((v) => v.key as keyof AllAppConfig)
		);
		return { appConfig: parseConfigList(data), envOverrides };
	};

	update = async (appConfig: Partial<AllAppConfig>) => {
		// Convert all values to string, stringifying JSON where needed
		const appConfigConvertedToString: Record<string, string> = {};
		for (const key in appConfig) {
			const value = (appConfig as any)[key];
			appConfigConvertedToString[key] =
				typeof value === 'object' && value !== null ? JSON.stringify(value) : String(value);
		}
		const res = await this.api.patch(
			'/admin/application-configuration',
			appConfigConvertedToString
		);
		return parseConfigList(res.data);
	};

	updateFavicon = async (favicon: File) => {
		const formData = new FormData();
		formData.append('file', favicon);

		await this.api.put(`/admin/application-assets/favicon`, formData);
	};

	updateLogo = async (logo: File, light = true) => {
		const formData = new FormData();
		formData.append('file', logo);

		await this.api.put(`/admin/application-assets/logo`, formData, {
			params: { light }
		});
		cachedApplicationLogo.bustCache(light);
	};

	deleteLogo = async (light = true) => {
		await this.api.delete(`/admin/application-assets/logo`, {
			params: { light }
		});
		cachedApplicationLogo.bustCache(light);
	};

	updateEmailLogo = async (emailLogo: File) => {
		const formData = new FormData();
		formData.append('file', emailLogo);

		await this.api.put(`/admin/application-assets/email`, formData);
		cachedEmailLogo.bustCache();
	};

	updateDefaultProfilePicture = async (defaultProfilePicture: File) => {
		const formData = new FormData();
		formData.append('file', defaultProfilePicture);

		await this.api.put(`/admin/application-assets/default-profile-picture`, formData);
		cachedDefaultProfilePicture.bustCache();
	};

	updateBackgroundImage = async (backgroundImage: File) => {
		const formData = new FormData();
		formData.append('file', backgroundImage!);

		await this.api.put(`/admin/application-assets/background`, formData);
		cachedBackgroundImage.bustCache();
	};

	deleteBackgroundImage = async () => {
		await this.api.delete(`/admin/application-assets/background`);
		cachedBackgroundImage.bustCache();
	};

	deleteDefaultProfilePicture = async () => {
		await this.api.delete('/admin/application-assets/default-profile-picture');
		cachedDefaultProfilePicture.bustCache();
		cachedProfilePicture.bustCache(get(userStore)!.id);
	};

	updateFont = async (font: File) => {
		const formData = new FormData();
		formData.append('file', font);

		await this.api.put('/admin/application-assets/font', formData);
		cachedTitleFont.bustCache();
	};

	deleteFont = async () => {
		await this.api.delete('/admin/application-assets/font');
		cachedTitleFont.bustCache();
	};

	sendTestEmail = async () => {
		await this.api.post('/admin/application-configuration/test-email');
	};

	syncLdap = async () => {
		await this.api.post('/admin/application-configuration/sync-ldap');
	};
}

function parseConfigList(data: AppConfigRawResponse) {
	const appConfig: Partial<AllAppConfig> = {};
	data.forEach(({ key, value }) => {
		(appConfig as any)[key] = parseValue(value);
	});

	return appConfig as AllAppConfig;
}

function parseValue(value: string) {
	// Try to parse JSON first
	try {
		const parsed = JSON.parse(value);
		if (typeof parsed === 'object' && parsed !== null) {
			return parsed;
		}
		value = String(parsed);
	} catch {
		// Keep the original string when the value is not valid JSON
	}

	// Handle rest of the types
	if (value === 'true') {
		return true;
	} else if (value === 'false') {
		return false;
	} else if (/^-?\d+(\.\d+)?$/.test(value)) {
		return parseFloat(value);
	} else {
		return value;
	}
}
