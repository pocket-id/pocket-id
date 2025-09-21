import BadgeCheckIcon from '@lucide/svelte/icons/badge-check';
import BadgeXIcon from '@lucide/svelte/icons/badge-x';
import CircleFadingArrowUp from '@lucide/svelte/icons/circle-fading-arrow-up';
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

export const imageUpdateFilters = [
	{
		value: true,
		label: 'Has updates',
		icon: CircleFadingArrowUp
	},
	{
		value: false,
		label: 'No updates',
		icon: BadgeXIcon
	}
];
