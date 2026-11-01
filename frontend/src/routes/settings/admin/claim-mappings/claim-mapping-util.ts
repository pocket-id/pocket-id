import { m } from '#lib/paraglide/messages.js';
import type { OidcUserField } from '#lib/types/oidc-claim-mapping-policy.type.ts';

export const claimsCount = (count: number) =>
	count === 1 ? m.claims_count_one() : m.claims_count({ count });

export const clientsCount = (count: number) =>
	count === 1 ? m.clients_count_one() : m.clients_count({ count });

export const policyHref = (id: string) => `/settings/admin/claim-mappings/${id}`;

export function userFieldLabel(field: OidcUserField) {
	const labels: Record<OidcUserField, () => string> = {
		id: m.user_id,
		email: m.email,
		email_verified: m.email_verified,
		first_name: m.first_name,
		last_name: m.last_name,
		full_name: m.full_name,
		display_name: m.display_name,
		username: m.username,
		locale: m.locale,
		picture: m.profile_picture,
		groups: m.groups
	};
	return labels[field]();
}
