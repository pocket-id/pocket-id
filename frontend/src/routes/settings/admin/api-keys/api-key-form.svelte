<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import type { ApiKeyCreate } from '$lib/types/api-key.type';
	import { createForm } from '$lib/utils/form-util';
	import { z } from 'zod';

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
		expiresAt: defaultExpiry.toISOString().slice(0, 16) // Format as YYYY-MM-DDTHH:MM for input
	};

	// Define a schema that validates the date is in the future
	const formSchema = z.object({
		name: z
			.string()
			.min(3, 'Name must be at least 3 characters')
			.max(50, 'Name cannot exceed 50 characters'),
		description: z.string().optional(),
		expiresAt: z
			.string()
			.refine((val) => new Date(val) > new Date(), {
				message: 'Expiration date must be in the future'
			})
			.transform((val) => new Date(val)) // Transform string to Date after validation
	});

	const { inputs, ...form } = createForm<typeof formSchema>(formSchema, apiKey);

	async function onSubmit() {
		const data = form.validate();
		if (!data) return;

		// Now we can trust that data.expiresAt is already in the future
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

<form on:submit|preventDefault={onSubmit}>
	<div class="grid grid-cols-1 items-start gap-5 md:grid-cols-2">
		<FormInput label="Name" bind:input={$inputs.name} description="Name to identify this API key" />
		<FormInput
			label="Expires At"
			type="datetime-local"
			description="When this API key will expire"
			bind:input={$inputs.expiresAt}
		/>
		<div class="col-span-1 md:col-span-2">
			<FormInput
				label="Description"
				description="Optional description to help identify this key's purpose"
				bind:input={$inputs.description}
			/>
		</div>
	</div>
	<div class="mt-5 flex justify-end">
		<Button {isLoading} type="submit">Generate API Key</Button>
	</div>
</form>
