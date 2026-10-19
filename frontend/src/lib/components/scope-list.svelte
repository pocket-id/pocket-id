<script lang="ts">
	import * as Item from '#lib/components/ui/item/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import type { InteractionScopeInfo } from '#lib/types/oidc.type.ts';
	import { LucideKeyRound, LucideMail, LucideUser, LucideUsers } from '@lucide/svelte';
	import ScopeItem from './scope-item.svelte';

	let {
		scopes,
		scopeInfo = [],
		selectedScopes = $bindable()
	}: {
		scopes?: string[] | null;
		scopeInfo?: InteractionScopeInfo[] | null;
		// When bound, every displayed scope gets a checkbox so the user can deselect it
		selectedScopes?: string[];
	} = $props();

	const standardScopes = ['openid', 'profile', 'email', 'groups', 'offline_access'];
	const infoByKey = $derived(new Map((scopeInfo || []).map((info) => [info.key, info])));
	const customScopes = $derived((scopes || []).filter((scope) => !standardScopes.includes(scope)));

	function selectionProps(scope: string) {
		if (!selectedScopes) {
			return {};
		}

		return {
			selected: selectedScopes.includes(scope),
			onSelectedChange: (selected: boolean) => {
				const others = (selectedScopes ?? []).filter((s) => s !== scope);
				selectedScopes = selected ? [...others, scope] : others;
			}
		};
	}
</script>

<Item.Group data-testid="scopes" class="gap-1">
	{#if (scopes || []).includes('email')}
		<ScopeItem
			icon={LucideMail}
			name={m.email()}
			description={m.view_your_email_address()}
			{...selectionProps('email')}
		/>
	{/if}
	{#if (scopes || []).includes('profile')}
		<ScopeItem
			icon={LucideUser}
			name={m.profile()}
			description={m.view_your_profile_information()}
			{...selectionProps('profile')}
		/>
	{/if}
	{#if (scopes || []).includes('groups')}
		<ScopeItem
			icon={LucideUsers}
			name={m.groups()}
			description={m.view_the_groups_you_are_a_member_of()}
			{...selectionProps('groups')}
		/>
	{/if}
	{#each customScopes as scope (scope)}
		<ScopeItem
			icon={LucideKeyRound}
			name={infoByKey.get(scope)?.name ?? scope}
			description={infoByKey.get(scope)?.description || m.access_an_api_on_your_behalf()}
			{...selectionProps(scope)}
		/>
	{/each}
</Item.Group>
