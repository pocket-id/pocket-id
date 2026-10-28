import { m } from '#lib/paraglide/messages.js';
import type { Feature } from '#lib/types/feature.type.ts';
import {
	LucideBookUser,
	LucideFileCode,
	LucideKey,
	LucideRefreshCw,
	LucideServer,
	LucideTags,
	LucideWaypoints,
	type LucideIcon
} from '@lucide/svelte';

export type FeatureRow = {
	id: Feature;
	title: string;
	description: string;
	// Where the feature shows up, which the Features tab adds to the description
	adds: string;
	inUse: string;
	icon: LucideIcon;
};

// The Features tab, the welcome step and the command palette list the features in this order
export function featureSections(appName: string): {
	title: string;
	description: string;
	rows: FeatureRow[];
}[] {
	return [
		{
			title: m.features_directory(),
			description: m.features_directory_description(),
			rows: [
				{
					id: 'ldap',
					title: m.ldap(),
					description: m.feature_ldap_description(),
					adds: m.feature_ldap_adds(),
					inUse: m.feature_ldap_in_use(),
					icon: LucideBookUser
				},
				{
					id: 'scim',
					title: m.scim_provisioning(),
					description: m.feature_scim_description(),
					adds: m.feature_scim_adds(),
					inUse: m.feature_scim_in_use(),
					icon: LucideRefreshCw
				},
				{
					id: 'customClaims',
					title: m.custom_claims(),
					description: m.feature_custom_claims_description(),
					adds: m.feature_custom_claims_adds(),
					inUse: m.feature_custom_claims_in_use(),
					icon: LucideTags
				}
			]
		},
		{
			title: m.features_clients(),
			description: m.features_clients_description({ appName }),
			rows: [
				{
					id: 'cimd',
					title: m.client_id_metadata_documents(),
					description: m.feature_cimd_description(),
					adds: m.feature_cimd_adds(),
					inUse: m.feature_cimd_in_use(),
					icon: LucideFileCode
				},
				{
					id: 'federatedCredentials',
					title: m.federated_client_credentials(),
					description: m.feature_federated_credentials_description(),
					adds: m.feature_federated_credentials_adds(),
					inUse: m.feature_federated_credentials_in_use(),
					icon: LucideWaypoints
				}
			]
		},
		{
			title: m.features_developers(),
			description: m.features_developers_description({ appName }),
			rows: [
				{
					id: 'apis',
					title: m.apis(),
					description: m.feature_apis_description(),
					adds: m.feature_apis_adds(),
					inUse: m.feature_apis_in_use(),
					icon: LucideServer
				},
				{
					id: 'apiKeys',
					title: m.api_keys(),
					description: m.feature_api_keys_description({ appName }),
					adds: m.feature_api_keys_adds(),
					inUse: m.feature_api_keys_in_use(),
					icon: LucideKey
				}
			]
		}
	];
}
