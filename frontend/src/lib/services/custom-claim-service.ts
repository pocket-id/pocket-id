import type { CustomClaim } from '#lib/types/custom-claim.type.ts';
import APIService from './api-service';
import { pathSegment } from '#lib/utils/path-segment-util.ts';

export default class CustomClaimService extends APIService {
	getSuggestions = async () => {
		const res = await this.api.get('/admin/custom-claims/suggestions');
		return res.data as string[];
	};

	updateUserCustomClaims = async (userId: string, claims: CustomClaim[]) => {
		const res = await this.api.put(`/admin/custom-claims/user/${pathSegment(userId)}`, claims);
		return res.data as CustomClaim[];
	};

	updateUserGroupCustomClaims = async (userGroupId: string, claims: CustomClaim[]) => {
		const res = await this.api.put(
			`/admin/custom-claims/user-group/${pathSegment(userGroupId)}`,
			claims
		);
		return res.data as CustomClaim[];
	};
}
