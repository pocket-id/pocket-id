<script lang="ts">
	import * as Item from '$lib/components/ui/item/index.js';
	import { m } from '$lib/paraglide/messages';
	import type { InteractionScopeInfo } from '$lib/types/oidc.type';
	import { LucideKeyRound, LucideMail, LucideUser, LucideUsers } from '@lucide/svelte';
	import ScopeItem from './scope-item.svelte';

	let {
		scopes,
		scopeInfo = []
	}: { scopes?: string[] | null; scopeInfo?: InteractionScopeInfo[] | null } = $props();

	const standardScopes = ['openid', 'profile', 'email', 'groups', 'offline_access'];
	const infoByKey = $derived(new Map((scopeInfo || []).map((info) => [info.key, info])));
	const customScopes = $derived((scopes || []).filter((scope) => !standardScopes.includes(scope)));
</script>

<Item.Group data-testid="scopes" class="gap-1">
	{#if (scopes || []).includes('email')}
		<ScopeItem icon={LucideMail} name={m.email()} description={m.view_your_email_address()} />
	{/if}
	{#if (scopes || []).includes('profile')}
		<ScopeItem
			icon={LucideUser}
			name={m.profile()}
			description={m.view_your_profile_information()}
		/>
	{/if}
	{#if (scopes || []).includes('groups')}
		<ScopeItem
			icon={LucideUsers}
			name={m.groups()}
			description={m.view_the_groups_you_are_a_member_of()}
		/>
	{/if}
	{#each customScopes as scope}
		<ScopeItem
			icon={LucideKeyRound}
			name={infoByKey.get(scope)?.name ?? scope}
			description={infoByKey.get(scope)?.description || m.access_an_api_on_your_behalf()}
		/>
	{/each}
</Item.Group>
