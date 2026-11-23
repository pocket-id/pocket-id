import OidcService from '#lib/services/oidc-service.ts';
import type { EndSessionParams } from '#lib/types/oidc.type.ts';
import { getAxiosErrorMessage } from '#lib/utils/error-util.ts';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ url }) => {
	const params: EndSessionParams = {
		idTokenHint: url.searchParams.get('id_token_hint') ?? '',
		clientId: url.searchParams.get('client_id') ?? '',
		postLogoutRedirectUri: url.searchParams.get('post_logout_redirect_uri') ?? '',
		state: url.searchParams.get('state') ?? ''
	};

	try {
		const { clientName } = await new OidcService().getEndSessionRequest(params);
		return { params, clientName, error: null };
	} catch (e) {
		return { params, clientName: null, error: getAxiosErrorMessage(e) };
	}
};
