import type { ScimServiceProvider, ScimServiceProviderCreate } from '#lib/types/scim.type.ts';
import APIService from './api-service';

class ScimService extends APIService {
	syncServiceProvider = async (serviceProviderId: string) => {
		return await this.api.post(`/admin/scim/service-provider/${serviceProviderId}/sync`);
	};

	createServiceProvider = async (serviceProvider: ScimServiceProviderCreate) => {
		return (await this.api.post('/admin/scim/service-provider', serviceProvider))
			.data as ScimServiceProvider;
	};

	updateServiceProvider = async (
		serviceProviderId: string,
		serviceProvider: ScimServiceProviderCreate
	) => {
		return (
			await this.api.put(`/admin/scim/service-provider/${serviceProviderId}`, serviceProvider)
		).data as ScimServiceProvider;
	};

	deleteServiceProvider = async (serviceProviderId: string) => {
		await this.api.delete(`/admin/scim/service-provider/${serviceProviderId}`);
	};
}

export default ScimService;
