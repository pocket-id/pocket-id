import APIService from './api-service';

export default class WarningService extends APIService {
	list = async () => {
		const response = await this.api.get('/admin/warnings');
		return response.data.warnings as string[];
	};
}
