<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import { m } from '$lib/paraglide/messages';
	import type { UserSignUp } from '$lib/types/user.type';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { tryCatch } from '$lib/utils/try-catch-util';
	import { z } from 'zod/v4';
	import appConfigStore from '$lib/stores/application-configuration-store';

	let {
		callback,
		isLoading
	}: {
		callback: (user: UserSignUp) => Promise<boolean>;
		isLoading: boolean;
	} = $props();

	const initialData: UserSignUp = {
		firstName: '',
		lastName: '',
		email: '',
		username: ''
	};

	const usernameRegex = $derived(
		$appConfigStore.allowUppercaseUsernames ? /^[a-zA-Z0-9_@.-]+$/ : /^[a-z0-9_@.-]+$/
	);
	const formSchema = $derived(
		z.object({
			firstName: z.string().min(1).max(50),
			lastName: z.string().max(50).optional(),
			username: z.string().min(2).max(30).regex(usernameRegex, m.username_can_only_contain()),
			email: z.email()
		})
	);
	type FormSchema = typeof formSchema;
	
	const { inputs, ...form } = $derived(createForm<FormSchema>(formSchema, initialData));

	let userData: UserSignUp | null = $state(null);

	async function onSubmit() {
		const data = form.validate();
		if (!data) return;

		isLoading = true;
		const result = await tryCatch(callback(data));
		if (result.data) {
			userData = data;
			isLoading = false;
		}
	}
</script>

<form id="sign-up-form" onsubmit={preventDefault(onSubmit)} class="w-full">
	<div class="mt-7 space-y-4">
		<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
			<FormInput label={m.first_name()} bind:input={$inputs.firstName} />
			<FormInput label={m.last_name()} bind:input={$inputs.lastName} />
		</div>

		<FormInput label={m.username()} bind:input={$inputs.username} />
		<FormInput label={m.email()} bind:input={$inputs.email} type="email" />
	</div>
</form>
