import { m } from '#lib/paraglide/messages.js';
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

export type NavItem = { href: string; label: string; icon: Component };
export type NavGroup = { label: string; items: NavItem[] };

// The sidebar, the rail and the phone bar share these destinations
export function navGroups({
	isAdmin,
	verificationPending,
	showApis,
	showApiKeys
}: {
	isAdmin: boolean;
	verificationPending: boolean;
	showApis: boolean;
	showApiKeys: boolean;
}): NavGroup[] {
	// A user who still has to verify their email can only reach their account until they do
	if (verificationPending) {
		return [
			{
				label: m.account(),
				items: [{ href: '/settings/account', label: m.my_account(), icon: IdCard }]
			}
		];
	}

	const groups: NavGroup[] = [
		{
			label: m.account(),
			items: [
				{ href: '/settings/account', label: m.my_account(), icon: IdCard },
				{ href: '/settings/apps', label: m.my_apps(), icon: LayoutGrid },
				{ href: '/settings/audit-log', label: m.activity(), icon: History }
			]
		}
	];

	if (isAdmin) {
		groups.push({
			label: m.administration(),
			items: [
				{ href: '/settings/admin/users', label: m.users(), icon: User },
				{ href: '/settings/admin/user-groups', label: m.groups(), icon: UsersRound },
				{ href: '/settings/admin/oidc-clients', label: m.oidc_clients(), icon: Blocks },
				{ href: '/settings/admin/apis', label: m.apis(), icon: Server },
				{ href: '/settings/admin/api-keys', label: m.api_keys(), icon: Key },
				{ href: '/settings/audit-log/global', label: m.audit_log(), icon: FileText },
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

// APIs and API keys join the sidebar once one exists (docs/design-system.md, PO11), and a failed check keeps the row
export async function checkApiRows() {
	const first = { pagination: { page: 1, limit: 1 } };
	const exists = (list: Promise<Paginated<unknown>>) =>
		list.then((r) => r.data.length > 0).catch(() => true);
	const [apis, apiKeys] = await Promise.all([
		exists(new ApisService().list(first)),
		exists(new ApiKeyService().list(first))
	]);
	return { apis, apiKeys };
}

// The most specific destination wins, so the global audit log is not also marked as Activity
export function activeHref(groups: NavGroup[], pathname: string) {
	return groups
		.flatMap((g) => g.items.map((i) => i.href))
		.filter((href) => pathname === href || pathname.startsWith(href + '/'))
		.sort((a, b) => b.length - a.length)[0];
}
