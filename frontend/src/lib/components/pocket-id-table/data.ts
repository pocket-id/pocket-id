import BadgeCheckIcon from '@lucide/svelte/icons/badge-check';
import CircleCheck from '@lucide/svelte/icons/circle-check';
import { m } from '$lib/paraglide/messages';

export const disabledFilters = [
	{
		value: false,
		label: m.enabled(),
		icon: CircleCheck
	},
	{
		value: true,
		label: m.disabled(),
		icon: BadgeCheckIcon
	}
];

export const userRoleFilters = [
	{
		value: true,
		label: m.admin(),
		icon: CircleCheck
	},
	{
		value: false,
		label: m.user(),
		icon: BadgeCheckIcon
	}
];
