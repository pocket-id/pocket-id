<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import SwitchWithLabel from '$lib/components/form/switch-with-label.svelte';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { User, UserCreate } from '$lib/types/user.type';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { emptyToUndefined } from '$lib/utils/zod-util';
	import { z } from 'zod/v4';

	let {
		callback,
		existingUser
	}: {
		existingUser?: User;
		callback: (user: UserCreate) => Promise<boolean>;
	} = $props();

	let isLoading = $state(false);
	let inputDisabled = $derived(!!existingUser?.ldapId && $appConfigStore.ldapEnabled);
	let hasManualDisplayNameEdit = $state(!!existingUser?.displayName);

	const user = {
		firstName: existingUser?.firstName || '',
		lastName: existingUser?.lastName || '',
		displayName: existingUser?.displayName || '',
		email: existingUser?.email || '',
		username: existingUser?.username || '',
		isAdmin: existingUser?.isAdmin || false,
		disabled: existingUser?.disabled || false
	};

	const formSchema = z.object({
		firstName: z.string().min(1).max(50),
		lastName: emptyToUndefined(z.string().max(50).optional()),
		displayName: z.string().max(100),
		username: z
			.string()
			.min(2)
			.max(30)
			.regex(/^[a-z0-9_@.-]+$/, m.username_can_only_contain()),
		email: z.email(),
		isAdmin: z.boolean(),
		disabled: z.boolean()
	});
	type FormSchema = typeof formSchema;

	const { inputs, ...form } = createForm<FormSchema>(formSchema, user);
	async function onSubmit() {
		const data = form.validate();
		if (!data) return;
		isLoading = true;
		const success = await callback(data);
		// Reset form if user was successfully created
		if (success && !existingUser) form.reset();
		isLoading = false;
	}
	function onNameInput() {
		if (!hasManualDisplayNameEdit) {
			$inputs.displayName.value = `${$inputs.firstName.value}${
				$inputs.lastName?.value ? ' ' + $inputs.lastName.value : ''
			}`;
		}
	}
</script>

<form onsubmit={preventDefault(onSubmit)}>
	<fieldset disabled={inputDisabled}>
		<div class="grid grid-cols-1 items-start gap-5 md:grid-cols-2">
			<FormInput label={m.first_name()} oninput={onNameInput} bind:input={$inputs.firstName} />
			<FormInput label={m.last_name()} oninput={onNameInput} bind:input={$inputs.lastName} />
			<FormInput label={m.display_name()} bind:input={$inputs.displayName} />
			<FormInput label={m.username()} bind:input={$inputs.username} />
			<FormInput label={m.email()} bind:input={$inputs.email} />
		</div>
		<div class="mt-5 grid grid-cols-1 items-start gap-5 md:grid-cols-2">
			<SwitchWithLabel
				id="admin-privileges"
				label={m.admin_privileges()}
				description={m.admins_have_full_access_to_the_admin_panel()}
				bind:checked={$inputs.isAdmin.value}
			/>
			<SwitchWithLabel
				id="user-disabled"
				label={m.user_disabled()}
				description={m.disabled_users_cannot_log_in_or_use_services()}
				bind:checked={$inputs.disabled.value}
			/>
		</div>
		<div class="mt-5 flex justify-end">
			<Button {isLoading} type="submit">{m.save()}</Button>
		</div>
	</fieldset>
</form>
