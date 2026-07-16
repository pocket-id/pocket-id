import type {
	DeviceLoginDecision,
	DeviceLoginExchangeResult,
	DeviceLoginRequest,
	DeviceLoginVerificationInfo
} from '$lib/types/device-login.type';
import APIService from './api-service';

export default class DeviceLoginService extends APIService {
	createRequest = async () => {
		const response = await this.api.post('/device-login/requests');
		return response.data as DeviceLoginRequest;
	};

	exchangeRequest = async (requestId: string): Promise<DeviceLoginExchangeResult> => {
		const response = await this.api.post(`/device-login/requests/${requestId}/exchange`);
		return response.status === 202 ? null : response.data;
	};

	inspectRequest = async (code: string) => {
		const response = await this.api.post('/device-login/verification', { code });
		return response.data as DeviceLoginVerificationInfo;
	};

	decideRequest = async (code: string, decision: DeviceLoginDecision) => {
		await this.api.post('/device-login/verification/decision', { code, decision });
	};
}
