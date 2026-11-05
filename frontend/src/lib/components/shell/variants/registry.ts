import type { Layout } from '#lib/components/appearance/appearance.svelte.ts';
import type { AppVersionInformation } from '#lib/types/application-configuration.type.ts';
import type { Component, Snippet } from 'svelte';
import type { NavGroup } from '../nav-items.ts';
import Backdrop from './backdrop.svelte';
import Classic from './classic.svelte';
import TwinCards from './twin-cards.svelte';

// Every shell frames the same <main> content, which the settings layout passes as children
export type ShellProps = {
	groups: NavGroup[];
	versionInformation: AppVersionInformation;
	showUpdate: boolean;
	children: Snippet;
};

export const shells: Record<Layout, Component<ShellProps>> = {
	classic: Classic,
	'twin-cards': TwinCards,
	backdrop: Backdrop
};
