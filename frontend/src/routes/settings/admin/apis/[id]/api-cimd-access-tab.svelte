<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import SwitchWithLabel from '$lib/components/form/switch-with-label.svelte';
	import { Button } from '$lib/components/ui/button';
	import Checkbox from '$lib/components/ui/checkbox/checkbox.svelte';
	import { Label } from '$lib/components/ui/label';
	import { m } from '$lib/paraglide/messages';
	import type { Api, ApiCimdAccessUpdate } from '$lib/types/api.type';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { z } from 'zod/v4';

	let { api, onSave }: { api: Api; onSave: (update: ApiCimdAccessUpdate) => Promise<void> } =
		$props();

	const formSchema = z.object({ enabled: z.boolean(), permissionIds: z.array(z.string()) });
	const { inputs, ...form } = createForm(formSchema, {
		enabled: api.allowCimdClients,
		permissionIds: api.permissions.filter((p) => p.allowedForCimdClients).map((p) => p.id)
	});

	async function save() {
		const data = form.validate();
		if (data) await onSave(data);
	}
</script>

<form novalidate onsubmit={preventDefault(save)}>
	<FormInput bind:input={$inputs.enabled} class="my-5">
		<SwitchWithLabel
			id="allow-cimd-clients"
			label={m.allow_all_metadata_document_clients()}
			description={m.allow_all_metadata_document_clients_description()}
			bind:checked={$inputs.enabled.value}
		/>
	</FormInput>

	{#if $inputs.enabled.value}
		<div class="mt-6">
			{#if api.permissions.length > 0}
				<FormInput bind:input={$inputs.permissionIds}>
					<Label class="mb-3">{m.granted_permissions()}</Label>
					<div class="flex flex-col gap-3">
						{#each api.permissions as permission (permission.id)}
							<div class="flex items-start gap-2">
								<Checkbox
									id={`cimd-permission-${permission.id}`}
									class="mt-0.5"
									checked={$inputs.permissionIds.value.includes(permission.id)}
									onCheckedChange={(checked: boolean) =>
										form.setValue(
											'permissionIds',
											checked
												? [...$inputs.permissionIds.value, permission.id]
												: $inputs.permissionIds.value.filter((id) => id !== permission.id)
										)}
								/>
								<div class="grid gap-1 leading-none">
									<Label
										for={`cimd-permission-${permission.id}`}
										class="mb-0 text-sm leading-none font-medium"
									>
										{permission.name}
									</Label>
									<p class="text-muted-foreground font-mono text-xs">{permission.key}</p>
								</div>
							</div>
						{/each}
					</div>
				</FormInput>
			{/if}
		</div>
	{/if}

	<div class="mt-5 flex justify-end">
		<Button type="submit" usePromiseLoading>{m.save()}</Button>
	</div>
</form>
