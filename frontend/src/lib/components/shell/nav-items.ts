import { m } from '#lib/paraglide/messages.js';
import type { Features } from '#lib/types/feature.type.ts';
import AnimatedIdCard from '#lib/icons/id-card-animated.svelte';
import AnimatedLayoutGrid from '#lib/icons/layout-grid-animated.svelte';
import AnimatedHistory from '@jis3r/icons/icons/history';
import AnimatedUser from '@jis3r/icons/icons/user';
import AnimatedUsersRound from '@jis3r/icons/icons/users-round';
import AnimatedBlocks from '@jis3r/icons/icons/blocks';
import AnimatedKey from '@jis3r/icons/icons/key';
import AnimatedFileText from '@jis3r/icons/icons/file-text';
import AnimatedSlidersHorizontal from '@jis3r/icons/icons/sliders-horizontal';
import type { IconProps } from '@jis3r/icons';
import {
	Blocks,
	FileText,
	History,
	IdCard,
	Key,
	LayoutGrid,
	Server,
	SlidersHorizontal,
	User,
	UsersRound
} from '@lucide/svelte';
import type { Component } from 'svelte';

// animated is the sidebar's moving version of the icon; My account and My apps use #lib/icons because the library has no id-card and its layout-grid motion was too busy
export type NavItem = {
	href: string;
	label: string;
	icon: Component;
	animated?: Component<IconProps>;
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
						icon: IdCard,
						animated: AnimatedIdCard
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
					icon: IdCard,
					animated: AnimatedIdCard
				},
				{
					href: '/settings/apps',
					label: m.my_apps(),
					icon: LayoutGrid,
					animated: AnimatedLayoutGrid
				},
				{
					href: '/settings/audit-log',
					label: m.activity(),
					icon: History,
					animated: AnimatedHistory
				}
			]
		}
	];

	if (isAdmin) {
		groups.push({
			label: m.administration(),
			items: [
				{ href: '/settings/admin/users', label: m.users(), icon: User, animated: AnimatedUser },
				{
					href: '/settings/admin/user-groups',
					label: m.groups(),
					icon: UsersRound,
					animated: AnimatedUsersRound
				},
				{
					href: '/settings/admin/oidc-clients',
					label: m.oidc_clients(),
					icon: Blocks,
					animated: AnimatedBlocks
				},
				...(features?.apis.enabled
					? [{ href: '/settings/admin/apis', label: m.apis(), icon: Server }]
					: []),
				...(features?.apiKeys.enabled
					? [
							{
								href: '/settings/admin/api-keys',
								label: m.api_keys(),
								icon: Key,
								animated: AnimatedKey
							}
						]
					: []),
				{
					href: '/settings/audit-log/global',
					label: m.audit_log(),
					icon: FileText,
					animated: AnimatedFileText
				},
				{
					href: '/settings/admin/application-configuration',
					label: m.configuration(),
					icon: SlidersHorizontal,
					animated: AnimatedSlidersHorizontal
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
