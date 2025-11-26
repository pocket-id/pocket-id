<script lang="ts">
	import CustomClaimsInput from '$lib/components/form/custom-claims-input.svelte';
	import SearchableMultiSelect from '$lib/components/form/searchable-multi-select.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Field from '$lib/components/ui/field';
	import * as Select from '$lib/components/ui/select';
	import { m } from '$lib/paraglide/messages';
	import UserGroupService from '$lib/services/user-group-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { AllAppConfig } from '$lib/types/application-configuration';
	import { debounced } from '$lib/utils/debounce-util';
	import { preventDefault } from '$lib/utils/event-util';
	import { onMount } from 'svelte';
	import { toast } from 'svelte-sonner';

	let {
		appConfig,
		callback
	}: {
		appConfig: AllAppConfig;
		callback: (updatedConfig: Partial<AllAppConfig>) => Promise<void>;
	} = $props();

	const userGroupService = new UserGroupService();

	let userGroups = $state<{ value: string; label: string }[]>([]);
	let selectedGroups = $state<{ value: string; label: string }[]>([]);
	let customClaims = $state(appConfig.signupDefaultCustomClaims || []);
	let allowUserSignups = $state(appConfig.allowUserSignups);
	let isLoading = $state(false);
	let isUserSearchLoading = $state(false);

	const signupOptions = {
		disabled: {
			label: m.disabled(),
			description: m.signup_disabled_description()
		},
		withToken: {
			label: m.signup_with_token(),
			description: m.signup_with_token_description()
		},
		open: {
			label: m.signup_open(),
			description: m.signup_open_description()
		}
	};

	async function loadUserGroups(search?: string) {
		userGroups = (await userGroupService.list({ search })).data.map((group) => ({
			value: group.id,
			label: group.name
		}));

		// Ensure selected groups are still in the list
		for (const selectedGroup of selectedGroups) {
			if (!userGroups.some((g) => g.value === selectedGroup.value)) {
				userGroups.push(selectedGroup);
			}
		}
	}

	async function loadSelectedGroups() {
		selectedGroups = (
			await Promise.all(
				appConfig.signupDefaultUserGroupIDs.map((groupId) => userGroupService.get(groupId))
			)
		).map((group) => ({
			value: group.id,
			label: group.name
		}));
	}

	const onUserGroupSearch = debounced(
		async (search: string) => await loadUserGroups(search),
		300,
		(loading) => (isUserSearchLoading = loading)
	);

	async function onSubmit() {
		isLoading = true;
		await callback({
			allowUserSignups: allowUserSignups,
			signupDefaultUserGroupIDs: selectedGroups.map((g) => g.value),
			signupDefaultCustomClaims: customClaims
		});
		toast.success(m.user_creation_updated_successfully());
		isLoading = false;
	}

	$effect(() => {
		loadSelectedGroups();
		customClaims = appConfig.signupDefaultCustomClaims || [];
		allowUserSignups = appConfig.allowUserSignups;
	});

	onMount(() => loadUserGroups());
</script>

<form onsubmit={preventDefault(onSubmit)}>
	<fieldset class="flex flex-col gap-5" disabled={$appConfigStore.uiConfigDisabled}>
		<Field.Field>
			<Field.Label for="enable-user-signup">{m.enable_user_signups()}</Field.Label>
			<Field.Description>
				{m.enable_user_signups_description()}
			</Field.Description>
			<Select.Root
				type="single"
				value={allowUserSignups}
				onValueChange={(v) => (allowUserSignups = v as typeof allowUserSignups)}
			>
				<Select.Trigger
					id="enable-user-signup"
					class="w-full"
					aria-label={m.enable_user_signups()}
					placeholder={m.enable_user_signups()}
				>
					{signupOptions[allowUserSignups]?.label}
				</Select.Trigger>
				<Select.Content>
					<Select.Item value="disabled">
						<div class="flex flex-col items-start gap-1">
							<span class="font-medium">{signupOptions.disabled.label}</span>
							<span class="text-muted-foreground text-xs">
								{signupOptions.disabled.description}
							</span>
						</div>
					</Select.Item>
					<Select.Item value="withToken">
						<div class="flex flex-col items-start gap-1">
							<span class="font-medium">{signupOptions.withToken.label}</span>
							<span class="text-muted-foreground text-xs">
								{signupOptions.withToken.description}
							</span>
						</div>
					</Select.Item>
					<Select.Item value="open">
						<div class="flex flex-col items-start gap-1">
							<span class="font-medium">{signupOptions.open.label}</span>
							<span class="text-muted-foreground text-xs">
								{signupOptions.open.description}
							</span>
						</div>
					</Select.Item>
				</Select.Content>
			</Select.Root>
		</Field.Field>

		<Field.Field>
			<Field.Label for="default-groups">{m.user_groups()}</Field.Label>
			<Field.Description>
				{m.user_creation_groups_description()}
			</Field.Description>
			<SearchableMultiSelect
				id="default-groups"
				items={userGroups}
				oninput={(e) => onUserGroupSearch(e.currentTarget.value)}
				selectedItems={selectedGroups.map((g) => g.value)}
				onSelect={(selected) => {
					selectedGroups = userGroups.filter((g) => selected.includes(g.value));
				}}
				isLoading={isUserSearchLoading}
				disableInternalSearch
			/>
		</Field.Field>
		<Field.Field>
			<Field.Label>{m.custom_claims()}</Field.Label>
			<Field.Description>
				{m.user_creation_claims_description()}
			</Field.Description>
			<CustomClaimsInput bind:customClaims />
		</Field.Field>

		<div class="flex justify-end pt-2">
			<Button {isLoading} type="submit">{m.save()}</Button>
		</div>
	</fieldset>
</form>
