import type { CustomClaim } from '$lib/types/custom-claim.type';
import APIService from './api-service';

export default class CustomClaimService extends APIService {
	getSuggestions = async () => {
		const res = await this.api.get('/custom-claims/suggestions');
		return res.data as string[];
	};

	updateUserCustomClaims = async (userId: string, claims: CustomClaim[]) => {
		const res = await this.api.put(`/custom-claims/user/${userId}`, claims);
		return res.data as CustomClaim[];
	};

	updateUserGroupCustomClaims = async (userGroupId: string, claims: CustomClaim[]) => {
		const res = await this.api.put(`/custom-claims/user-group/${userGroupId}`, claims);
		return res.data as CustomClaim[];
	};
}
