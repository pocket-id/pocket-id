import { version as currentVersion } from '$app/environment';
import APIService from './api-service';

export default class VersionService extends APIService {
	getNewestVersion = async () => {
		const response = await this.api
			.get('/version/latest', { timeout: 2000 })
			.then((res) => res.data);
		return response.latestVersion;
	};

	getCurrentVersion = () => currentVersion;
}
