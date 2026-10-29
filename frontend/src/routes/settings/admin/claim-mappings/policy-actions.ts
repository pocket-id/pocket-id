import { goto } from '$app/navigation';
import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
import { m } from '#lib/paraglide/messages.js';
import OidcClaimMappingPolicyService from '#lib/services/oidc-claim-mapping-policy-service.ts';
import type { OidcClaimMappingPolicyMetadataDto } from '#lib/types/oidc-claim-mapping-policy.type.ts';
import { axiosErrorToast } from '#lib/utils/error-util.ts';
import { toast } from 'svelte-sonner';
import { policyHref } from './claim-mapping-util';

const policyService = new OidcClaimMappingPolicyService();

type PolicyRef = Pick<OidcClaimMappingPolicyMetadataDto, 'id' | 'name' | 'isDefault'>;

// Duplicating copies every claim into a new policy and opens it, which is how the built-in policy gets customized
export async function duplicatePolicy(policy: PolicyRef) {
	try {
		const source = await policyService.getClaimMappingPolicy(policy.id);
		const copy = await policyService.createClaimMappingPolicy({
			name: m.name_copy({ name: source.name }),
			isDefault: false,
			claimMappings: source.claimMappings
		});
		toast.success(m.policy_duplicated(), {
			description: m.name_has_the_same_claims({ name: copy.name })
		});
		await goto(policyHref(copy.id));
	} catch (e) {
		axiosErrorToast(e);
	}
}

// The update replaces the whole policy, so the saved claims are read first and sent back unchanged
export async function makeDefaultPolicy(policy: PolicyRef) {
	try {
		const saved = await policyService.getClaimMappingPolicy(policy.id);
		await policyService.updateClaimMappingPolicy(policy.id, { ...saved, isDefault: true });
		toast.success(m.name_is_now_the_default({ name: policy.name }), {
			description: m.apps_without_a_policy_use_it()
		});
		return true;
	} catch (e) {
		axiosErrorToast(e);
		return false;
	}
}

export async function confirmDeletePolicy(policy: PolicyRef, onDeleted: () => unknown) {
	let assigned: number;
	try {
		const clients = await policyService.listClients(policy.id, {
			pagination: { page: 1, limit: 1 }
		});
		assigned = clients.pagination.totalItems;
	} catch (e) {
		axiosErrorToast(e);
		return;
	}

	const consequence =
		assigned === 0
			? m.delete_unused_policy_consequence()
			: assigned === 1
				? m.delete_policy_consequence_one()
				: m.delete_policy_consequence({ count: assigned });

	openConfirmDialog({
		title: m.delete_name_question({ name: policy.name }),
		message: policy.isDefault
			? `${consequence} ${m.built_in_policy_becomes_the_default()}`
			: consequence,
		confirm: {
			label: m.delete_policy(),
			destructive: true,
			action: async () => {
				try {
					await policyService.deleteClaimMappingPolicy(policy.id);
					toast.success(m.name_deleted({ name: policy.name }));
					await onDeleted();
				} catch (e) {
					axiosErrorToast(e);
				}
			}
		}
	});
}
