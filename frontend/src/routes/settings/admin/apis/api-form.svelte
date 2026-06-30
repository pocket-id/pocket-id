<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import type { Api, ApiCreate } from '$lib/types/api.type';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { z } from 'zod/v4';

	let {
		callback,
		existingApi
	}: {
		existingApi?: Api;
		callback: (api: ApiCreate) => Promise<boolean>;
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

	const { inputs, ...form } = createForm<FormSchema>(formSchema, api);

	async function onSubmit() {
		const data = form.validate();
		if (!data) return;
		isLoading = true;
		const success = await callback(data);
		if (success && !existingApi) {
			form.reset();
		}
		isLoading = false;
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
	<div class="mt-5 flex justify-end">
		<Button {isLoading} type="submit">{m.save()}</Button>
	</div>
</form>
