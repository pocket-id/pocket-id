<script lang="ts">
	import CheckboxWithLabel from '$lib/components/form/checkbox-with-label.svelte';
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { User, UserCreate } from '$lib/types/user.type';
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
		isAdmin: existingUser?.isAdmin || false
	};

	const formSchema = z.object({
		firstName: z.string().min(1).max(50),
		lastName: z.string().min(1).max(50),
		username: z
			.string()
			.min(2)
			.max(30)
			.regex(
				/^[a-z0-9_@.-]+$/,
				"Username can only contain lowercase letters, numbers, underscores, dots, hyphens, and '@' symbols"
			),
		email: z.string().email(),
		isAdmin: z.boolean()
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

<form onsubmit={onSubmit}>
	<fieldset disabled={inputDisabled}>
		<div class="grid grid-cols-1 items-start gap-5 md:grid-cols-2">
			<FormInput label="First name" bind:input={$inputs.firstName} />
			<FormInput label="Last name" bind:input={$inputs.lastName} />
			<FormInput label="Username" bind:input={$inputs.username} />
			<FormInput label="Email" bind:input={$inputs.email} />
			<CheckboxWithLabel
				id="admin-privileges"
				label="Admin Privileges"
				description="Admins have full access to the admin panel."
				bind:checked={$inputs.isAdmin.value}
			/>
		</div>
		<div class="mt-5 flex justify-end">
			<Button {isLoading} type="submit">Save</Button>
		</div>
	</fieldset>
</form>
