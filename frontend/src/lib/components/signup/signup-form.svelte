<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import { m } from '$lib/paraglide/messages';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { UserSignUp } from '$lib/types/user.type';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { tryCatch } from '$lib/utils/try-catch-util';
	import { emptyToUndefined, usernameSchema } from '$lib/utils/zod-util';
	import { get } from 'svelte/store';
	import { z } from 'zod/v4';

	let {
		callback,
		isLoading,
		requiredEmailDomain
	}: {
		callback: (user: UserSignUp) => Promise<boolean>;
		isLoading: boolean;
		requiredEmailDomain?: string | null;
	} = $props();

	const initialData: UserSignUp = {
		firstName: '',
		lastName: '',
		email: '',
		username: ''
	};

	const emailSchema = requiredEmailDomain
		? z.email().refine((v) => v.toLowerCase().endsWith(`@${requiredEmailDomain.toLowerCase()}`), {
				message: m.email_must_use_domain({ domain: `@${requiredEmailDomain}` })
			})
		: get(appConfigStore).requireUserEmail
			? z.email()
			: emptyToUndefined(z.email().optional());

	const formSchema = z.object({
		firstName: z.string().max(50),
		lastName: emptyToUndefined(z.string().max(50).optional()),
		username: usernameSchema,
		email: emailSchema
	});
	type FormSchema = typeof formSchema;

	const { inputs, ...form } = createForm<FormSchema>(formSchema, initialData);

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
		<FormInput label={m.username()} bind:input={$inputs.username} />
		<FormInput
			label={m.email()}
			bind:input={$inputs.email}
			type="email"
			placeholder={requiredEmailDomain ? `you@${requiredEmailDomain}` : undefined}
			description={requiredEmailDomain
				? m.email_domain_required_hint({ domain: `@${requiredEmailDomain}` })
				: undefined}
		/>

		<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
			<FormInput label={m.first_name()} bind:input={$inputs.firstName} />
			<FormInput label={m.last_name()} bind:input={$inputs.lastName} />
		</div>
	</div>
</form>
