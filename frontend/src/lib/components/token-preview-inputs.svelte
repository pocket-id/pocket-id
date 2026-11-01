<script lang="ts">
	import MultiSelect from '#lib/components/form/multi-select.svelte';
	import SearchableSelect from '#lib/components/form/searchable-select.svelte';
	import * as Field from '#lib/components/ui/field/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import UserService from '#lib/services/user-service.ts';
	import type { User } from '#lib/types/user.type.ts';
	import { debounced } from '#lib/utils/debounce-util.ts';

	let {
		userId = $bindable(),
		scopes = $bindable()
	}: {
		userId: string;
		scopes: string[];
	} = $props();

	const userService = new UserService();

	let users = $state<User[]>([]);
	let isUserSearchLoading = $state(false);

	async function loadUsers(search?: string) {
		users = (await userService.list({ search, pagination: { limit: 10, page: 1 } })).data;
		userId ||= users[0]?.id ?? '';
	}

	const onUserSearch = debounced(
		(search: string) => loadUsers(search),
		300,
		(loading) => (isUserSearchLoading = loading)
	);

	$effect(() => {
		loadUsers();
	});
</script>

<!-- The fields sit side by side only when there is room, so the same inputs fit both a sheet and a narrow sidebar -->
<div class="@container">
	<div class="grid gap-4 @md:grid-cols-2">
		<Field.Field>
			<Field.Label>{m.person()}</Field.Label>
			<SearchableSelect
				selectText={m.select_user()}
				isLoading={isUserSearchLoading}
				items={users.map((user) => ({ value: user.id, label: user.displayName || user.username }))}
				bind:value={userId}
				oninput={(e) => onUserSearch(e.currentTarget.value)}
			/>
		</Field.Field>
		<Field.Field>
			<Field.Label>{m.scopes()}</Field.Label>
			<MultiSelect
				placeholder={m.none()}
				items={['openid', 'email', 'profile', 'groups'].map((scope) => ({
					value: scope,
					label: scope
				}))}
				bind:selectedItems={scopes}
			/>
		</Field.Field>
	</div>
</div>
