import type { Passkey } from '$lib/types/passkey.type';
import type { User } from '$lib/types/user.type';
import APIService from './api-service';
import userStore from '$lib/stores/user-store';
import type { AuthenticationResponseJSON, RegistrationResponseJSON } from '@simplewebauthn/browser';
class WebAuthnService extends APIService {
  getRegistrationOptions = async () => (await this.api.get(`/webauthn/register/start`)).data;

  finishRegistration = async (body: RegistrationResponseJSON) =>
    (await this.api.post(`/webauthn/register/finish`, body)).data as Passkey;

  getLoginOptions = async () => (await this.api.get(`/webauthn/login/start`)).data;

  finishLogin = async (body: AuthenticationResponseJSON) =>
    (await this.api.post(`/webauthn/login/finish`, body)).data as User;

  logout = async () => {
    await this.api.post(`/webauthn/logout`);
    userStore.clearUser();
  };

  listCredentials = async () => (await this.api.get(`/webauthn/credentials`)).data as Passkey[];

  removeCredential = async (id: string) => {
    await this.api.delete(`/webauthn/credentials/${id}`);
  };

  updateCredentialName = async (id: string, name: string) => {
    await this.api.patch(`/webauthn/credentials/${id}`, { name });
  };

  reauthenticate = async (body?: AuthenticationResponseJSON) => {
    const res = await this.api.post('/webauthn/reauthenticate', body);
    return res.data.reauthenticationToken as string;
  };
}

export default WebAuthnService;
