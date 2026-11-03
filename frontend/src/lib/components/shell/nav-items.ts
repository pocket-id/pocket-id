import { m } from '#lib/paraglide/messages.js';
import type { Features } from '#lib/types/feature.type.ts';
import Blocks from '~icons/solar/blocks-bold-duotone';
import Braces from '~icons/solar/code-bold-duotone';
import FileText from '~icons/solar/document-text-bold-duotone';
import History from '~icons/solar/history-bold-duotone';
import IdCard from '~icons/solar/user-id-bold-duotone';
import LayoutGrid from '~icons/solar/widget-bold-duotone';
import Server from '~icons/solar/server-bold-duotone';
import SlidersHorizontal from '~icons/solar/tuning-2-bold-duotone';
import User from '~icons/solar/user-bold-duotone';
import UsersRound from '~icons/solar/users-group-rounded-bold-duotone';
import type { Component } from 'svelte';

export type NavItem = {
	href: string;
	label: string;
	icon: Component;
};
export type NavGroup = { label: string; items: NavItem[] };

// The sidebar, the rail and the phone bar share these destinations
export function navGroups({
	isAdmin,
	verificationPending,
	features
}: {
	isAdmin: boolean;
	verificationPending: boolean;
	features: Features | null;
}): NavGroup[] {
	// A user who still has to verify their email can only reach their account until they do
	if (verificationPending) {
		return [
			{
				label: m.account(),
				items: [
					{
						href: '/settings/account',
						label: m.my_account(),
						icon: IdCard
					}
				]
			}
		];
	}

	const groups: NavGroup[] = [
		{
			label: m.account(),
			items: [
				{
					href: '/settings/account',
					label: m.my_account(),
					icon: IdCard
				},
				{
					href: '/settings/apps',
					label: m.my_apps(),
					icon: LayoutGrid
				},
				{
					href: '/settings/audit-log',
					label: m.activity(),
					icon: History
				}
			]
		}
	];

	if (isAdmin) {
		groups.push({
			label: m.administration(),
			items: [
				{ href: '/settings/admin/users', label: m.users(), icon: User },
				{
					href: '/settings/admin/user-groups',
					label: m.groups(),
					icon: UsersRound
				},
				{
					href: '/settings/admin/oidc-clients',
					label: m.oidc_clients(),
					icon: Blocks
				},
				...(features?.claimMappings.enabled
					? [{ href: '/settings/admin/claim-mappings', label: m.claim_mappings(), icon: Braces }]
					: []),
				...(features?.apis.enabled
					? [{ href: '/settings/admin/apis', label: m.apis(), icon: Server }]
					: []),
				{
					href: '/settings/audit-log/global',
					label: m.audit_log(),
					icon: FileText
				},
				{
					href: '/settings/admin/application-configuration',
					label: m.configuration(),
					icon: SlidersHorizontal
				}
			]
		});
	}

	return groups;
}

// The most specific destination wins, so the global audit log is not also marked as Activity
export function activeHref(groups: NavGroup[], pathname: string) {
	return groups
		.flatMap((g) => g.items.map((i) => i.href))
		.filter((href) => pathname === href || pathname.startsWith(href + '/'))
		.sort((a, b) => b.length - a.length)[0];
}
