<script lang="ts">
	import { goto } from '$app/navigation';
	import FormInput from '#lib/components/form/form-input.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Sheet from '#lib/components/ui/sheet/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import UserGroupService from '#lib/services/user-group-service.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { preventDefault } from '#lib/utils/event-util.ts';
	import { createForm } from '#lib/utils/form-util.ts';
	import { untrack } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { groupNameSchema } from './group-name-schema';

	let { open = $bindable() }: { open: boolean } = $props();

	const userGroupService = new UserGroupService();
	const formId = $props.id();

	let isLoading = $state(false);
	// The claim name follows the friendly name until it is edited by hand
	let nameEdited = false;

	const formStore = createForm(groupNameSchema, { friendlyName: '', name: '' });
	const { inputs } = formStore;

	$effect(() => {
		if (open) return;
		untrack(() => {
			formStore.reset();
			nameEdited = false;
		});
	});

	function onFriendlyNameInput() {
		if (nameEdited) return;
		$inputs.name.value = $inputs.friendlyName.value.toLowerCase().replace(/[^a-z0-9_]/g, '_');
	}

	async function onSubmit() {
		const data = formStore.validate();
		if (!data) return;
		isLoading = true;
		try {
			const group = await userGroupService.create(data);
			open = false;
			toast.success(m.name_added({ name: group.friendlyName }));
			goto(`/settings/admin/user-groups/${group.id}`);
		} catch (e) {
			axiosErrorToast(e);
		} finally {
			isLoading = false;
		}
	}
</script>

<Sheet.Root bind:open>
	<Sheet.Content title={m.add_group()}>
		<form id={formId} onsubmit={preventDefault(onSubmit)} class="flex flex-col gap-6">
			<FormInput
				label={m.friendly_name()}
				placeholder={m.group_friendly_name_placeholder()}
				bind:input={$inputs.friendlyName}
				onInput={onFriendlyNameInput}
			/>
			<FormInput
				label={m.name()}
				placeholder={m.group_name_placeholder()}
				description={m.group_name_helper()}
				inputClass="font-mono"
				bind:input={$inputs.name}
				onInput={() => (nameEdited = true)}
			/>
		</form>
		{#snippet footer()}
			<Button variant="ghost" onclick={() => (open = false)}>{m.cancel()}</Button>
			<Button type="submit" form={formId} {isLoading}>{m.add_group()}</Button>
		{/snippet}
	</Sheet.Content>
</Sheet.Root>
