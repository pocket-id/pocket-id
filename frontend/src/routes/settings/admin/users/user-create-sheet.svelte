<script lang="ts">
	import { goto } from '$app/navigation';
	import FormInput from '#lib/components/form/form-input.svelte';
	import UserGroupInput from '#lib/components/form/user-group-input.svelte';
	import SettingRow from '#lib/components/setting-row.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { Checkbox } from '#lib/components/ui/checkbox/index.ts';
	import * as Sheet from '#lib/components/ui/sheet/index.ts';
	import { Switch } from '#lib/components/ui/switch/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import UserService from '#lib/services/user-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { preventDefault } from '#lib/utils/event-util.ts';
	import { createForm } from '#lib/utils/form-util.ts';
	import { emptyToUndefined, usernameSchema } from '#lib/utils/zod-util.ts';
	import { toast } from 'svelte-sonner';
	import { untrack } from 'svelte';
	import { get } from 'svelte/store';
	import { z } from 'zod/v4';

	let {
		open = $bindable(),
		emailsVerifiedPerDefault
	}: { open: boolean; emailsVerifiedPerDefault: boolean } = $props();

	const userService = new UserService();
	const formId = $props.id();

	let isLoading = $state(false);
	let groupIds = $state<string[]>([]);

	const formSchema = z.object({
		firstName: z.string().max(50),
		lastName: emptyToUndefined(z.string().max(50)),
		displayName: z.string().max(100),
		username: usernameSchema,
		email: get(appConfigStore).requireUserEmail ? z.email() : emptyToUndefined(z.email()),
		emailVerified: z.boolean(),
		isAdmin: z.boolean()
	});

	const formStore = createForm(formSchema, {
		firstName: '',
		lastName: '',
		displayName: '',
		username: '',
		email: '',
		emailVerified: emailsVerifiedPerDefault,
		isAdmin: false
	});
	const { inputs } = formStore;

	// The display name follows the first and last name until it is edited by hand
	let displayNameEdited = false;
	function onNameInput() {
		if (displayNameEdited) return;
		$inputs.displayName.value = [$inputs.firstName.value, $inputs.lastName?.value]
			.filter(Boolean)
			.join(' ')
			.trim();
	}

	// A closed sheet starts empty the next time it opens
	$effect(() => {
		if (open) return;
		untrack(() => {
			formStore.reset();
			groupIds = [];
			displayNameEdited = false;
		});
	});

	async function onSubmit() {
		const data = formStore.validate();
		if (!data) return;
		isLoading = true;
		try {
			const user = await userService.create({ ...data, disabled: false });
			if (groupIds.length) await userService.updateUserGroups(user.id, groupIds);
			open = false;
			toast.success(m.name_added({ name: user.displayName }));
			goto(`/settings/admin/users/${user.id}`);
		} catch (e) {
			axiosErrorToast(e);
		} finally {
			isLoading = false;
		}
	}
</script>

<Sheet.Root bind:open>
	<Sheet.Content title={m.add_user()}>
		<form id={formId} onsubmit={preventDefault(onSubmit)} class="flex flex-col gap-6" novalidate>
			<div class="grid gap-x-4 gap-y-6 sm:grid-cols-2">
				<FormInput
					label={m.first_name()}
					optional
					bind:input={$inputs.firstName}
					onInput={onNameInput}
				/>
				<FormInput
					label={m.last_name()}
					optional
					bind:input={$inputs.lastName}
					onInput={onNameInput}
				/>
			</div>
			<FormInput
				label={m.display_name()}
				optional
				description={m.display_name_follows_names()}
				bind:input={$inputs.displayName}
				onInput={() => (displayNameEdited = true)}
			/>
			<FormInput
				label={m.username()}
				description={m.username_rule()}
				bind:input={$inputs.username}
			/>
			<div class="flex flex-col gap-3">
				<FormInput
					label={m.email()}
					optional={!$appConfigStore.requireUserEmail}
					description={m.email_used_for_login_codes()}
					type="email"
					bind:input={$inputs.email}
				/>
				{#if $appConfigStore.emailVerification !== 'disabled'}
					<label class="text-control text-fg flex items-center gap-3 font-medium">
						<Checkbox bind:checked={$inputs.emailVerified.value} />
						{m.email_address_is_verified()}
					</label>
				{/if}
			</div>
			<section class="flex flex-col gap-2 border-t pt-6">
				<h3 class="text-title text-fg">{m.access()}</h3>
				<SettingRow
					title={m.admin_privileges()}
					description={m.admins_have_full_access_to_the_admin_panel()}
				>
					{#snippet control(props)}
						<Switch {...props} bind:checked={$inputs.isAdmin.value} />
					{/snippet}
				</SettingRow>
				<FormInput labelFor="default-groups" label={m.groups()}>
					<UserGroupInput bind:selectedGroupIds={groupIds} />
				</FormInput>
			</section>
		</form>
		{#snippet footer()}
			<Button variant="ghost" onclick={() => (open = false)}>{m.cancel()}</Button>
			<Button type="submit" form={formId} {isLoading}>{m.add_user()}</Button>
		{/snippet}
	</Sheet.Content>
</Sheet.Root>
