<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import type { ApiKeyCreate } from '$lib/types/api-key.type';
	import { createForm } from '$lib/utils/form-util';
	import { z } from 'zod';

	// Add a function to ensure the date is in the future
	function ensureFutureDate(date: Date): Date {
		const now = new Date();
		return date > now ? date : new Date(now.getTime() + 24 * 60 * 60 * 1000); // Default to tomorrow
	}

	let {
		callback
	}: {
		callback: (apiKey: ApiKeyCreate) => Promise<boolean>;
	} = $props();

	let isLoading = $state(false);

	// Set default expiration to 30 days from now
	const defaultExpiry = ensureFutureDate(new Date());
	defaultExpiry.setDate(defaultExpiry.getDate() + 30);

	const apiKey = {
		name: '',
		description: '',
		expiresAt: defaultExpiry.toISOString().slice(0, 16) // Format as YYYY-MM-DDTHH:MM for input
	};

	const formSchema = z.object({
		name: z.string().min(3).max(50),
		description: z.string().optional(),
		expiresAt: z.string().transform((val) => new Date(val)) // Transform string to Date
	});

	const { inputs, ...form } = createForm<typeof formSchema>(formSchema, apiKey);

	async function onSubmit() {
		const data = form.validate();
		if (!data) return;

		// Ensure the date is in the future
		const apiKeyData: ApiKeyCreate = {
			name: data.name,
			description: data.description,
			expiresAt: ensureFutureDate(new Date(data.expiresAt)) // Ensure it's a Date object
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
