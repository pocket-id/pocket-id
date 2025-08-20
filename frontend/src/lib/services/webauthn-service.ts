import type { Passkey } from '$lib/types/passkey.type';
import type { User } from '$lib/types/user.type';
import APIService from './api-service';
import userStore from '$lib/stores/user-store';
import type { AuthenticationResponseJSON, RegistrationResponseJSON } from '@simplewebauthn/browser';

class WebAuthnService extends APIService {
	async getRegistrationOptions() {
		return (await this.api.get(`/webauthn/register/start`)).data;
	}

	async finishRegistration(body: RegistrationResponseJSON) {
		return (await this.api.post(`/webauthn/register/finish`, body)).data as Passkey;
	}

	async getLoginOptions() {
		return (await this.api.get(`/webauthn/login/start`)).data;
	}

	async finishLogin(body: AuthenticationResponseJSON) {
		return (await this.api.post(`/webauthn/login/finish`, body)).data as User;
	}

	async logout() {
		await this.api.post(`/webauthn/logout`);
		userStore.clearUser();
	}

	async listCredentials() {
		return (await this.api.get(`/webauthn/credentials`)).data as Passkey[];
	}

	async removeCredential(id: string) {
		await this.api.delete(`/webauthn/credentials/${id}`);
	}

	async updateCredentialName(id: string, name: string) {
		await this.api.patch(`/webauthn/credentials/${id}`, { name });
	}

	async reauthenticate(body?: AuthenticationResponseJSON) {
		const res = await this.api.post('/webauthn/reauthenticate', body);
		return res.data.reauthenticationToken as string;
	}
}

export default WebAuthnService;
