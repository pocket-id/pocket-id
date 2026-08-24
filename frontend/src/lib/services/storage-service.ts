import APIService from './api-service';

export default class StorageService extends APIService {
	getSqliteStorageWarning = async () => {
		const response = await this.api.get('/storage/sqlite-warning').then((res) => res.data);
		return response.showWarning as boolean;
	};
}
