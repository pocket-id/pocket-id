<script lang="ts">
	import { m } from '#lib/paraglide/messages.js';
	import userStore from '#lib/stores/user-store.ts';
	import type { InteractionScopeInfo } from '#lib/types/oidc.type.ts';
	import { LucideKeyRound, LucideMail, LucideUserRound, LucideUsersRound } from '@lucide/svelte';
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

	const profileValue = $derived(
		$userStore
			? [$userStore.displayName, $userStore.username].filter(Boolean).join(' · ')
			: undefined
	);
	const groupsValue = $derived(
		$userStore
			? $userStore.userGroups.map((group) => group.friendlyName).join(', ') || m.no_groups_yet()
			: undefined
	);

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

<fieldset
	data-testid="scopes"
	class="bg-surface divide-border w-full divide-y overflow-hidden rounded-md border"
>
	<legend class="sr-only">{m.permissions()}</legend>
	{#if (scopes || []).includes('email')}
		<ScopeItem
			icon={LucideMail}
			value={$userStore?.email}
			category={m.email_address()}
			{...selectionProps('email')}
		/>
	{/if}
	{#if (scopes || []).includes('profile')}
		<ScopeItem
			icon={LucideUserRound}
			value={profileValue}
			category={m.name_username_and_picture()}
			{...selectionProps('profile')}
		/>
	{/if}
	{#if (scopes || []).includes('groups')}
		<ScopeItem
			icon={LucideUsersRound}
			value={groupsValue}
			category={m.group_membership()}
			{...selectionProps('groups')}
		/>
	{/if}
	{#each customScopes as scope (scope)}
		<ScopeItem
			icon={LucideKeyRound}
			value={infoByKey.get(scope)?.name ?? scope}
			category={infoByKey.get(scope)?.description || m.access_an_api_on_your_behalf()}
			{...selectionProps(scope)}
		/>
	{/each}
</fieldset>
