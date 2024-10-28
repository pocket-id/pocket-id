import { version as currentVersion } from '$app/environment';
import type { AllAppConfig, AppConfigRawResponse } from '$lib/types/application-configuration';
import axios from 'axios';
import APIService from './api-service';

export default class AppConfigService extends APIService {
	async list(showAll = false) {
		let url = '/application-configuration';
		if (showAll) {
			url += '/all';
		}

		const { data } = await this.api.get<AppConfigRawResponse>(url);

		const appConfig: Partial<AllAppConfig> = {};
		data.forEach(({ key, value }) => {
			(appConfig as any)[key] = this.parseValue(value);
		});

		return appConfig as AllAppConfig;
	}

	async update(appConfig: AllAppConfig) {
		// Convert all values to string
		const appConfigConvertedToString = {};
		for (const key in appConfig) {
			(appConfigConvertedToString as any)[key] = (appConfig as any)[key].toString();
		}
		const res = await this.api.put('/application-configuration', appConfigConvertedToString);
		return res.data as AllAppConfig;
	}

	async updateFavicon(favicon: File) {
		const formData = new FormData();
		formData.append('file', favicon!);

		await this.api.put(`/application-configuration/favicon`, formData);
	}

	async updateLogo(logo: File, light = true) {
		const formData = new FormData();
		formData.append('file', logo!);

		await this.api.put(`/application-configuration/logo`, formData, {
			params: { light }
		});
	}

	async updateBackgroundImage(backgroundImage: File) {
		const formData = new FormData();
		formData.append('file', backgroundImage!);

		await this.api.put(`/application-configuration/background-image`, formData);
	}

	async getVersionInformation() {
		const response = (
			await axios.get('https://api.github.com/repos/stonith404/pocket-id/releases/latest')
		).data;

		const newestVersion = response.tag_name.replace('v', '');
		const isUpToDate = newestVersion === currentVersion;

		return {
			isUpToDate,
			newestVersion,
			currentVersion
		};
	}

	private parseValue(value: string) {
		if (value === 'true') {
			return true;
		} else if (value === 'false') {
			return false;
		} else if (!isNaN(parseFloat(value))) {
			return parseFloat(value);
		} else {
			return value;
		}
	}
}
