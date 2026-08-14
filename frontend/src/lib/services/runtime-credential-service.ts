import type { RuntimeCredential } from '$lib/types/runtime-credential.type';
import APIService from './api-service';

export default class RuntimeCredentialService extends APIService {
	list = async () => {
		const res = await this.api.get('/runtime-credentials');
		return res.data as RuntimeCredential[];
	};

	listForUser = async (userId: string) => {
		const res = await this.api.get(`/users/${userId}/runtime-credentials`);
		return res.data as RuntimeCredential[];
	};

	updateName = async (credentialId: string, name: string) => {
		const res = await this.api.patch(`/runtime-credentials/${credentialId}`, { name });
		return res.data as RuntimeCredential;
	};

	revoke = async (credentialId: string) => {
		await this.api.delete(`/runtime-credentials/${credentialId}`);
	};

	revokeForUser = async (userId: string, credentialId: string) => {
		await this.api.delete(`/users/${userId}/runtime-credentials/${credentialId}`);
	};
}
