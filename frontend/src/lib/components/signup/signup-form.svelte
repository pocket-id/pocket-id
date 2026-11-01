<script lang="ts">
	import { authPrimaryClass } from '#lib/components/auth/auth-util.ts';
	import FormInput from '#lib/components/form/form-input.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { UserSignUp } from '#lib/types/user.type.ts';
	import { preventDefault } from '#lib/utils/event-util.ts';
	import { createForm } from '#lib/utils/form-util.ts';
	import { emptyToUndefined, usernameSchema } from '#lib/utils/zod-util.ts';
	import { get } from 'svelte/store';
	import { z } from 'zod/v4';

	let {
		callback,
		submitLabel
	}: {
		callback: (user: UserSignUp) => Promise<void>;
		submitLabel: string;
	} = $props();

	const initialData: UserSignUp = {
		firstName: '',
		lastName: '',
		email: '',
		username: ''
	};

	const formSchema = z.object({
		firstName: z.string().max(50),
		lastName: emptyToUndefined(z.string().max(50).optional()),
		username: usernameSchema,
		email: get(appConfigStore).requireUserEmail ? z.email() : emptyToUndefined(z.email().optional())
	});
	type FormSchema = typeof formSchema;

	const { inputs, ...form } = createForm<FormSchema>(formSchema, initialData);

	let isLoading = $state(false);

	async function onSubmit() {
		const data = form.validate();
		if (!data) return;

		isLoading = true;
		await callback(data);
		isLoading = false;
	}
</script>

<form onsubmit={preventDefault(onSubmit)} class="flex flex-col gap-4" novalidate>
	<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
		<FormInput
			label={m.first_name()}
			optional
			autocomplete="given-name"
			inputClass="h-(--h-xl)"
			bind:input={$inputs.firstName}
		/>
		<FormInput
			label={m.last_name()}
			optional
			autocomplete="family-name"
			inputClass="h-(--h-xl)"
			bind:input={$inputs.lastName}
		/>
	</div>
	<FormInput
		label={m.username()}
		description={m.username_rule()}
		autocomplete="username"
		inputClass="h-(--h-xl)"
		bind:input={$inputs.username}
	/>
	<FormInput
		label={m.email_address()}
		optional={!$appConfigStore.requireUserEmail}
		type="email"
		autocomplete="email"
		inputClass="h-(--h-xl)"
		bind:input={$inputs.email}
	/>
	<Button type="submit" class={authPrimaryClass + ' mt-2'} {isLoading}>{submitLabel}</Button>
</form>
