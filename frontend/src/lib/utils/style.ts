import { clsx, type ClassValue } from 'clsx';
import { extendTailwindMerge } from 'tailwind-merge';
import { defaultConfig } from 'tailwind-variants';

// The type scale defines text-* sizes that tailwind-merge would otherwise read as text colors and drop
const twMergeConfig = {
	extend: {
		classGroups: {
			'font-size': [
				{
					text: [
						'display-l',
						'display-m',
						'display-s',
						'title',
						'heading',
						'body-l',
						'body',
						'control',
						'caption',
						'micro',
						'mono',
						'mono-s',
						'code-cell'
					]
				}
			]
		}
	}
};

const twMerge = extendTailwindMerge(twMergeConfig);
defaultConfig.twMergeConfig = twMergeConfig;

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}

export function cssVar(name: string) {
	return getComputedStyle(document.documentElement).getPropertyValue(name).trim();
}

// Durations and holds come from the motion tokens, which collapse to zero under reduced motion
export function cssMs(name: string) {
	return parseFloat(cssVar(name)) || 0;
}

export type WithoutChild<T> = T extends { child?: any } ? Omit<T, 'child'> : T;
export type WithoutChildren<T> = T extends { children?: any } ? Omit<T, 'children'> : T;
export type WithoutChildrenOrChild<T> = WithoutChildren<WithoutChild<T>>;
export type WithElementRef<T, U extends HTMLElement = HTMLElement> = T & { ref?: U | null };
