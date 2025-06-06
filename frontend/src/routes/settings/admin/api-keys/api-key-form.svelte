<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import type { ApiKeyCreate } from '$lib/types/api-key.type';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { z } from 'zod/v4';

	let {
		callback
	}: {
		callback: (apiKey: ApiKeyCreate) => Promise<boolean>;
	} = $props();

	let isLoading = $state(false);

	// Set default expiration to 30 days from now
	const defaultExpiry = new Date();
	defaultExpiry.setDate(defaultExpiry.getDate() + 30);

	const apiKey = {
		name: '',
		description: '',
		expiresAt: defaultExpiry
	};

	const formSchema = z.object({
		name: z
			.string()
			.min(3, m.name_must_be_at_least_3_characters())
			.max(50, m.name_cannot_exceed_50_characters()),
		description: z.string().default(''),
		expiresAt: z.date().min(new Date(), m.expiration_date_must_be_in_the_future())
	});

	const { inputs, ...form } = createForm<typeof formSchema>(formSchema, apiKey);

	async function onSubmit() {
		const data = form.validate();
		if (!data) return;

		const apiKeyData: ApiKeyCreate = {
			name: data.name,
			description: data.description,
			expiresAt: data.expiresAt
		};

		isLoading = true;
		const success = await callback(apiKeyData);
		if (success) form.reset();
		isLoading = false;
	}
</script>

<form onsubmit={preventDefault(onSubmit)}>
	<div class="grid grid-cols-1 items-start gap-5 md:grid-cols-2">
		<FormInput
			label={m.name()}
			bind:input={$inputs.name}
			description={m.name_to_identify_this_api_key()}
		/>
		<FormInput
			label={m.expires_at()}
			type="date"
			description={m.when_this_api_key_will_expire()}
			bind:input={$inputs.expiresAt}
		/>
		<div class="col-span-1 md:col-span-2">
			<FormInput
				label={m.description()}
				description={m.optional_description_to_help_identify_this_keys_purpose()}
				bind:input={$inputs.description}
			/>
		</div>
	</div>
	<div class="mt-5 flex justify-end">
		<Button {isLoading} type="submit">{m.save()}</Button>
	</div>
</form>
