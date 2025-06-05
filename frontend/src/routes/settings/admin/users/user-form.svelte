<script lang="ts">
	import CheckboxWithLabel from '$lib/components/form/checkbox-with-label.svelte';
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { User, UserCreate } from '$lib/types/user.type';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { z } from 'zod';

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

	const formSchema = z.object({
		firstName: z.string().min(1).max(50),
		lastName: z.string().max(50),
		username: z
			.string()
			.min(2)
			.max(30)
			.regex(/^[a-z0-9_@.-]+$/, m.username_can_only_contain()),
		email: z.string().email(),
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
</script>

<form onsubmit={preventDefault(onSubmit)}>
	<fieldset disabled={inputDisabled}>
		<div class="grid grid-cols-1 items-start gap-5 md:grid-cols-2">
			<FormInput label={m.first_name()} bind:input={$inputs.firstName} />
			<FormInput label={m.last_name()} bind:input={$inputs.lastName} />
			<FormInput label={m.username()} bind:input={$inputs.username} />
			<FormInput label={m.email()} bind:input={$inputs.email} />
			<CheckboxWithLabel
				id="admin-privileges"
				label={m.admin_privileges()}
				description={m.admins_have_full_access_to_the_admin_panel()}
				bind:checked={$inputs.isAdmin.value}
			/>
			<CheckboxWithLabel
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
