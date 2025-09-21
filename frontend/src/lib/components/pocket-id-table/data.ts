import BadgeCheckIcon from '@lucide/svelte/icons/badge-check';
import BadgeXIcon from '@lucide/svelte/icons/badge-x';
import CircleFadingArrowUp from '@lucide/svelte/icons/circle-fading-arrow-up';
import CircleCheck from '@lucide/svelte/icons/circle-check';

// Replaced missing translation keys with plain strings
export const usageFilters = [
	{
		value: true,
		label: 'In use',
		icon: BadgeCheckIcon
	},
	{
		value: false,
		label: 'Unused',
		icon: CircleCheck
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
