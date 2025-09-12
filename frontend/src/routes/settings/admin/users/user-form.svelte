<script lang="ts">
	import SwitchWithLabel from '$lib/components/form/switch-with-label.svelte';
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import type { User, UserCreate } from '$lib/types/user.type';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
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

	const user = {
		firstName: existingUser?.firstName || '',
		lastName: existingUser?.lastName || '',
		email: existingUser?.email || '',
		username: existingUser?.username || '',
		isAdmin: existingUser?.isAdmin || false,
		disabled: existingUser?.disabled || false
	};

	const usernameRegex = /^[a-zA-Z0-9_@.-]+$/;
	const formSchema = $derived(
		z.object({
			firstName: z.string().min(1).max(50),
			lastName: z.string().max(50),
			username: z.string().min(2).max(30).regex(usernameRegex, m.username_can_only_contain()),
			email: z.email(),
			isAdmin: z.boolean(),
			disabled: z.boolean()
		})
	);
	type FormSchema = typeof formSchema;
	
	const { inputs, ...form } = $derived(createForm<FormSchema>(formSchema, user));
	async function onSubmit() {
		const data = form.validate();
		if (!data) return;
		// Normalize username to lowercase before submitting to backend
		data.username = data.username.toLowerCase();
		isLoading = true;
		const success = await callback(data);
		// Reset form if user was successfully created
		if (success && !existingUser) form.reset();
		isLoading = false;
	}
</script>

<form onsubmit={preventDefault(onSubmit)}>
	<fieldset disabled={inputDisabled}>
		<div class="grid grid-cols-1 items-start gap-5 md:grid-cols-2">
			<FormInput label={m.first_name()} bind:input={$inputs.firstName} />
			<FormInput label={m.last_name()} bind:input={$inputs.lastName} />
			<FormInput label={m.username()} bind:input={$inputs.username} />
			<FormInput label={m.email()} bind:input={$inputs.email} />
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
