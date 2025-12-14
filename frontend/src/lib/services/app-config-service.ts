import userStore from '$lib/stores/user-store';
import type { AllAppConfig, AppConfigRawResponse } from '$lib/types/application-configuration';
import {
	cachedApplicationLogo,
	cachedBackgroundImage,
	cachedDefaultProfilePicture,
	cachedEmailLogo,
	cachedProfilePicture
} from '$lib/utils/cached-image-util';
import { get } from 'svelte/store';
import APIService from './api-service';

export default class AppConfigService extends APIService {
	list = async (showAll = false) => {
		let url = '/application-configuration';
		if (showAll) url += '/all';
		const { data } = await this.api.get<AppConfigRawResponse>(url);
		return parseConfigList(data);
	};

	update = async (appConfig: AllAppConfig) => {
		// Convert all values to string, stringifying JSON where needed
		const appConfigConvertedToString: Record<string, string> = {};
		for (const key in appConfig) {
			const value = (appConfig as any)[key];
			appConfigConvertedToString[key] =
				typeof value === 'object' && value !== null ? JSON.stringify(value) : String(value);
		}
		const res = await this.api.put('/application-configuration', appConfigConvertedToString);
		return parseConfigList(res.data);
	};

	updateFavicon = async (favicon: File) => {
		const formData = new FormData();
		formData.append('file', favicon);

		await this.api.put(`/application-images/favicon`, formData);
	};

	updateLogo = async (logo: File, light = true) => {
		const formData = new FormData();
		formData.append('file', logo);

		await this.api.put(`/application-images/logo`, formData, {
			params: { light }
		});
		cachedApplicationLogo.bustCache(light);
	};

	updateEmailLogo = async (emailLogo: File) => {
		const formData = new FormData();
		formData.append('file', emailLogo);

		await this.api.put(`/application-images/email`, formData);
		cachedEmailLogo.bustCache();
	};

	updateDefaultProfilePicture = async (defaultProfilePicture: File) => {
		const formData = new FormData();
		formData.append('file', defaultProfilePicture);

		await this.api.put(`/application-images/default-profile-picture`, formData);
		cachedDefaultProfilePicture.bustCache();
	};

	updateBackgroundImage = async (backgroundImage: File) => {
		const formData = new FormData();
		formData.append('file', backgroundImage!);

		await this.api.put(`/application-images/background`, formData);
		cachedBackgroundImage.bustCache();
	};

	deleteDefaultProfilePicture = async () => {
		await this.api.delete('/application-images/default-profile-picture');
		cachedDefaultProfilePicture.bustCache();
		cachedProfilePicture.bustCache(get(userStore)!.id);
	};

	sendTestEmail = async () => {
		await this.api.post('/application-configuration/test-email');
	};

	syncLdap = async () => {
		await this.api.post('/application-configuration/sync-ldap');
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
	} catch {}

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
