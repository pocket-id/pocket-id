<script lang="ts">
	import CustomClaimsInput from '$lib/components/form/custom-claims-input.svelte';
	import MultiSelect from '$lib/components/form/multi-select.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Label } from '$lib/components/ui/label';
	import { Separator } from '$lib/components/ui/separator';
	import { m } from '$lib/paraglide/messages';
	import type { AllAppConfig } from '$lib/types/application-configuration';
	import type { UserGroup } from '$lib/types/user-group.type';
	import { preventDefault } from '$lib/utils/event-util';
  import { toast } from 'svelte-sonner';

	let {
		appConfig,
		callback,
		userGroups = []
	}: {
		appConfig: AllAppConfig;
		callback: (updatedConfig: Partial<AllAppConfig>) => Promise<void>;
		userGroups: UserGroup[];
	} = $props();

	let selectedGroupIds = $state(appConfig.signupDefaultUserGroupIDs || []);
	let customClaims = $state(appConfig.signupDefaultCustomClaims || []);
	let isLoading = $state(false);

	const groupItems = $derived(userGroups.map((g) => ({ value: g.id, label: g.friendlyName })));

	$effect(() => {
		selectedGroupIds = appConfig.signupDefaultUserGroupIDs || [];
		customClaims = appConfig.signupDefaultCustomClaims || [];
	});

	async function onSubmit() {
		isLoading = true;
		await callback({
			signupDefaultUserGroupIDs: selectedGroupIds,
			signupDefaultCustomClaims: customClaims
		});
		toast.success(m.signup_defaults_updated_successfully());
		isLoading = false;
	}
</script>

<form class="space-y-6" onsubmit={preventDefault(onSubmit)}>
	<div>
		<Label for="default-groups">{m.user_groups()}</Label>
		<p class="text-muted-foreground mt-1 mb-2 text-xs">
			{m.signup_defaults_groups_description()}
		</p>
		<MultiSelect
			items={groupItems}
			bind:selectedItems={selectedGroupIds}
			onSelect={() => (selectedGroupIds = selectedGroupIds)}
		/>
	</div>

	<Separator />

	<div>
		<Label>{m.custom_claims()}</Label>
		<p class="text-muted-foreground mt-1 mb-2 text-xs">
			{m.signup_defaults_claims_description()}
		</p>
		<CustomClaimsInput bind:customClaims={customClaims} />
	</div>

	<div class="flex justify-end pt-2">
		<Button {isLoading} type="submit">{m.save()}</Button>
	</div>
</form>
