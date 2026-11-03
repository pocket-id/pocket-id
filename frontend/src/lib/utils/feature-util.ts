import { m } from '#lib/paraglide/messages.js';
import type { Feature } from '#lib/types/feature.type.ts';
import BookUserIcon from '~icons/solar/notebook-bold-duotone';
import BracesIcon from '~icons/solar/code-bold-duotone';
import FileCodeIcon from '~icons/solar/code-square-bold-duotone';
import KeyIcon from '~icons/solar/key-bold-duotone';
import RefreshCwIcon from '~icons/solar/refresh-bold-duotone';
import ServerIcon from '~icons/solar/server-bold-duotone';
import TagsIcon from '~icons/solar/tag-bold-duotone';
import WaypointsIcon from '~icons/solar/routing-2-bold-duotone';
import type { Component } from 'svelte';

export type FeatureRow = {
	id: Feature;
	title: string;
	description: string;
	// Where the feature shows up, which the Features tab adds to the description
	adds: string;
	inUse: string;
	icon: Component;
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
					icon: BookUserIcon
				},
				{
					id: 'scim',
					title: m.scim_provisioning(),
					description: m.feature_scim_description(),
					adds: m.feature_scim_adds(),
					inUse: m.feature_scim_in_use(),
					icon: RefreshCwIcon
				},
				{
					id: 'customClaims',
					title: m.custom_claims(),
					description: m.feature_custom_claims_description(),
					adds: m.feature_custom_claims_adds(),
					inUse: m.feature_custom_claims_in_use(),
					icon: TagsIcon
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
					icon: FileCodeIcon
				},
				{
					id: 'federatedCredentials',
					title: m.federated_client_credentials(),
					description: m.feature_federated_credentials_description(),
					adds: m.feature_federated_credentials_adds(),
					inUse: m.feature_federated_credentials_in_use(),
					icon: WaypointsIcon
				},
				{
					id: 'claimMappings',
					title: m.claim_mappings(),
					description: m.feature_claim_mappings_description(),
					adds: m.feature_claim_mappings_adds(),
					inUse: m.feature_claim_mappings_in_use(),
					icon: BracesIcon
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
					icon: ServerIcon
				},
				{
					id: 'apiKeys',
					title: m.api_keys(),
					description: m.feature_api_keys_description({ appName }),
					adds: m.feature_api_keys_adds(),
					inUse: m.feature_api_keys_in_use(),
					icon: KeyIcon
				}
			]
		}
	];
}
