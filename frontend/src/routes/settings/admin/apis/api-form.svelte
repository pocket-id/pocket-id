<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import type { Api, ApiCreate } from '$lib/types/api.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { trackFormChanges } from '$lib/utils/unsaved-changes-util.svelte';
	import { z } from 'zod/v4';

	let {
		callback,
		existingApi
	}: {
		existingApi?: Api;
		callback: (api: ApiCreate) => Promise<void>;
	} = $props();

	let isLoading = $state(false);
	const isEdit = !!existingApi;

	const api = {
		name: existingApi?.name || '',
		resource: existingApi?.resource || ''
	};

	const formSchema = z.object({
		name: z.string().min(1).max(50),
		resource: z
			.url()
			.min(1)
			.max(350)
			.refine((value) => !/[#\s]/.test(value), {
				message: 'Resource must not include whitespace or a fragment'
			})
	});
	type FormSchema = typeof formSchema;

	const formStore = createForm<FormSchema>(formSchema, api);
	const { inputs } = formStore;

	async function saveApi(data: z.infer<FormSchema>) {
		await callback(data);
		if (!existingApi) formStore.reset();
	}

	// Create mode has its own Save button rather than going through the unsaved-changes bar.
	async function onSubmit() {
		const data = formStore.validate();
		if (!data) return;
		isLoading = true;
		try {
			await saveApi(data);
		} catch (e) {
			axiosErrorToast(e);
		} finally {
			isLoading = false;
		}
	}

	if (isEdit) {
		trackFormChanges(() => formStore, saveApi);
	}
</script>

<form onsubmit={preventDefault(onSubmit)}>
	<div class="flex flex-col gap-3">
		<FormInput label={m.name()} bind:input={$inputs.name} />
		<FormInput
			label={m.api_resource()}
			description={m.api_resource_description()}
			bind:input={$inputs.resource}
			readonly={isEdit}
		/>
	</div>
	{#if !isEdit}
		<div class="mt-5 flex justify-end">
			<Button {isLoading} type="submit">{m.save()}</Button>
		</div>
	{/if}
</form>
