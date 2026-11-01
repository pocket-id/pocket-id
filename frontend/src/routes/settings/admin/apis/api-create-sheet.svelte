<script lang="ts">
	import { goto } from '$app/navigation';
	import FormInput from '#lib/components/form/form-input.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Sheet from '#lib/components/ui/sheet/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import ApisService from '#lib/services/apis-service.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { preventDefault } from '#lib/utils/event-util.ts';
	import { createForm } from '#lib/utils/form-util.ts';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod/v4';

	let { open = $bindable() }: { open: boolean } = $props();

	const apisService = new ApisService();
	const formId = $props.id();
	let isLoading = $state(false);

	const formSchema = z.object({
		name: z.string().min(1).max(50),
		resource: z
			.url()
			.max(350)
			.refine((value) => !/[#\s]/.test(value), { message: m.resource_no_whitespace_or_fragment() })
	});
	const formStore = createForm(formSchema, { name: '', resource: '' });
	const { inputs } = formStore;

	async function onSubmit() {
		const data = formStore.validate();
		if (!data) return;
		isLoading = true;
		try {
			const api = await apisService.create(data);
			open = false;
			toast.success(m.name_added({ name: api.name }));
			goto(`/settings/admin/apis/${api.id}`);
		} catch (e) {
			axiosErrorToast(e);
		} finally {
			isLoading = false;
		}
	}
</script>

<Sheet.Root bind:open onOpenChangeComplete={(o) => !o && formStore.reset()}>
	<Sheet.Content title={m.add_api()}>
		<form id={formId} onsubmit={preventDefault(onSubmit)} class="flex flex-col gap-6">
			<FormInput
				label={m.name()}
				placeholder={m.api_name_placeholder()}
				description={m.api_name_helper()}
				bind:input={$inputs.name}
			/>
			<FormInput
				label={m.api_resource_uri()}
				placeholder="https://api.example.com"
				description={m.api_resource_description()}
				inputClass="font-mono"
				bind:input={$inputs.resource}
			/>
		</form>
		{#snippet footer()}
			<Button variant="ghost" onclick={() => (open = false)}>{m.cancel()}</Button>
			<Button type="submit" form={formId} {isLoading}>{m.add_api()}</Button>
		{/snippet}
	</Sheet.Content>
</Sheet.Root>
