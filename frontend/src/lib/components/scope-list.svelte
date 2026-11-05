<script lang="ts">
	import * as Alert from '@pocket-id/ui/alert';
	import { m } from '#lib/paraglide/messages.js';
	import type { InteractionScopeInfo } from '#lib/types/oidc.type.ts';
	import { isAdminEquivalentScope, scopeLabel } from '#lib/utils/api-key-scopes.ts';
	import KeyRoundIcon from '~icons/solar/key-minimalistic-bold-duotone';
	import MailIcon from '~icons/solar/letter-bold-duotone';
	import TriangleAlertIcon from '~icons/solar/danger-triangle-bold-duotone';
	import UserRoundIcon from '~icons/solar/user-rounded-bold-duotone';
	import UsersRoundIcon from '~icons/solar/users-group-rounded-bold-duotone';
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
	// Only Pocket ID's own scopes carry this risk, since a custom API's permission can share a key without meaning the same
	const grantsAdminEquivalent = $derived(
		(selectedScopes ?? scopes ?? []).some(
			(scope) => infoByKey.get(scope)?.builtIn && isAdminEquivalentScope(scope)
		)
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
			icon={MailIcon}
			name={m.email()}
			description={m.consent_email_description()}
			{...selectionProps('email')}
		/>
	{/if}
	{#if (scopes || []).includes('profile')}
		<ScopeItem
			icon={UserRoundIcon}
			name={m.profile()}
			description={m.consent_profile_description()}
			{...selectionProps('profile')}
		/>
	{/if}
	{#if (scopes || []).includes('groups')}
		<ScopeItem
			icon={UsersRoundIcon}
			name={m.groups()}
			description={m.consent_groups_description()}
			{...selectionProps('groups')}
		/>
	{/if}
	{#each customScopes as scope (scope)}
		{@const info = infoByKey.get(scope)}
		{@const label = info?.builtIn ? scopeLabel(scope) : info}
		<ScopeItem
			icon={KeyRoundIcon}
			name={label?.name ?? scope}
			description={label?.description || m.access_an_api_on_your_behalf()}
			adminAccess={info?.builtIn && isAdminEquivalentScope(scope)}
			{...selectionProps(scope)}
		/>
	{/each}
</fieldset>

{#if grantsAdminEquivalent}
	<Alert.Root variant="warning" class="mt-3">
		<TriangleAlertIcon />
		<Alert.Title>{m.app_can_gain_full_admin_access()}</Alert.Title>
		<Alert.Description>{m.app_can_gain_full_admin_access_description()}</Alert.Description>
	</Alert.Root>
{/if}
