<script lang="ts">
	import SettingRow from '#lib/components/setting-row.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import { Checkbox } from '#lib/components/ui/checkbox/index.ts';
	import { Switch } from '#lib/components/ui/switch/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import type { Api, ApiCimdAccessUpdate } from '#lib/types/api.type.ts';
	import { createForm } from '#lib/utils/form-util.ts';
	import { trackFormChanges } from '#lib/utils/unsaved-changes-util.svelte.ts';
	import { LucideFileText } from '@lucide/svelte';
	import { z } from 'zod/v4';

	let { api, onSave }: { api: Api; onSave: (update: ApiCimdAccessUpdate) => Promise<void> } =
		$props();

	const formSchema = z.object({ enabled: z.boolean(), permissionIds: z.array(z.string()) });
	const formStore = createForm(formSchema, {
		enabled: api.allowCimdClients,
		permissionIds: api.permissions.filter((p) => p.allowedForCimdClients).map((p) => p.id)
	});
	const { inputs } = formStore;

	trackFormChanges(() => formStore, onSave);
</script>

<SettingsSection
	title={m.metadata_document_clients()}
	description={m.metadata_document_clients_description()}
>
	<SettingRow
		title={m.allow_all_metadata_document_clients()}
		description={m.allow_all_metadata_document_clients_description()}
		icon={LucideFileText}
	>
		{#snippet control(props)}
			<Switch {...props} bind:checked={$inputs.enabled.value} />
		{/snippet}
	</SettingRow>
	{#if $inputs.enabled.value && api.permissions.length > 0}
		<fieldset class="mt-4 flex min-w-0 flex-col gap-2">
			<legend class="text-caption text-fg-muted mb-2 font-medium">{m.granted_permissions()}</legend>
			<div class="rounded-md border">
				{#each api.permissions as permission (permission.id)}
					<label
						class="flex min-h-(--row-2line) items-center gap-3 border-b px-4 py-2 last:border-b-0"
					>
						<Checkbox
							checked={$inputs.permissionIds.value.includes(permission.id)}
							onCheckedChange={(checked) =>
								formStore.setValue(
									'permissionIds',
									checked
										? [...$inputs.permissionIds.value, permission.id]
										: $inputs.permissionIds.value.filter((id) => id !== permission.id)
								)}
						/>
						<span class="flex min-w-0 flex-col">
							<span class="text-control text-fg font-medium">{permission.name}</span>
							<span class="text-mono-s text-fg-muted font-mono">{permission.key}</span>
						</span>
					</label>
				{/each}
			</div>
		</fieldset>
	{/if}
</SettingsSection>
