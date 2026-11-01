<script lang="ts">
	import * as Field from '#lib/components/ui/field/index.ts';
	import * as RadioGroup from '#lib/components/ui/radio-group/index.ts';
	import * as Tabs from '#lib/components/ui/tabs/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import {
		grantableScopes,
		levelOf,
		presetOf,
		readOnlyScopes,
		resourcesFor,
		withLevel,
		type ScopeLevel,
		type ScopePreset,
		type ScopeResource
	} from '#lib/utils/api-key-scopes.ts';
	import type { FormInput } from '#lib/utils/form-util.ts';
	import { untrack } from 'svelte';

	let { input = $bindable(), isAdmin }: { input: FormInput<string[]>; isAdmin: boolean } = $props();

	// The preset is kept apart from the scopes, so Custom stays open even while the picked scopes happen to match a preset
	let preset = $state<ScopePreset>(untrack(() => presetOf(input.value, isAdmin)));

	const presetDescriptions: Record<ScopePreset, () => string> = {
		full: m.api_key_full_access_description,
		read: m.api_key_read_only_description,
		custom: m.api_key_custom_access_description
	};

	const groups = $derived(
		[
			{ label: m.your_account(), resources: resourcesFor(isAdmin).filter((r) => !r.admin) },
			{ label: m.administration(), resources: resourcesFor(isAdmin).filter((r) => r.admin) }
		].filter((group) => group.resources.length > 0)
	);

	// Custom starts from nothing, so a key only gets what is picked on purpose
	function choosePreset(value: string) {
		preset = value as ScopePreset;
		if (preset === 'full') input.value = grantableScopes(isAdmin);
		if (preset === 'read') input.value = readOnlyScopes(isAdmin);
		if (preset === 'custom') input.value = [];
	}

	// Levels keep their column across rows, so a resource with one scope lines up under Read or Write
	function levelsOf(resource: ScopeResource) {
		const levels: { value: ScopeLevel; label: string; column: string }[] = [
			{ value: 'none', label: m.none(), column: 'col-start-1' }
		];
		if (resource.read) levels.push({ value: 'read', label: m.read(), column: 'col-start-2' });
		if (resource.write) {
			const label = resource.read ? m.write() : m.allow();
			levels.push({ value: 'write', label, column: 'col-start-3' });
		}
		return levels;
	}
</script>

<Field.Set class="gap-3">
	<!-- A legend is no flex item of its fieldset, so its margin sets the gap that Field puts between a label and its control -->
	<Field.Legend variant="label" class="mb-1.5">{m.permissions()}</Field.Legend>
	<div class="flex flex-col gap-1.5">
		<Tabs.Root bind:value={() => preset, choosePreset}>
			<Tabs.List aria-label={m.permissions()}>
				<Tabs.Trigger value="full">{m.full_access()}</Tabs.Trigger>
				<Tabs.Trigger value="read">{m.read_only()}</Tabs.Trigger>
				<Tabs.Trigger value="custom">{m.custom()}</Tabs.Trigger>
			</Tabs.List>
		</Tabs.Root>
		<Field.Description>{presetDescriptions[preset]()}</Field.Description>
	</div>

	{#if preset === 'custom'}
		{#each groups as group (group.label)}
			<section class="flex flex-col">
				<h3 class="text-caption text-fg-muted mt-2 mb-1 font-medium">{group.label}</h3>
				<ul class="divide-y border-y">
					{#each group.resources as resource (resource.id)}
						<li class="flex gap-x-4 gap-y-2 py-2.5 max-sm:flex-col sm:items-center">
							<div class="min-w-0 flex-1">
								<span id="scope-{resource.id}" class="text-control text-fg block font-medium">
									{resource.label()}
								</span>
								<span class="text-caption text-fg-muted block">{resource.description()}</span>
							</div>
							<RadioGroup.Root
								aria-labelledby="scope-{resource.id}"
								orientation="horizontal"
								class="grid w-56 shrink-0 grid-cols-3 gap-3"
								value={levelOf(resource, input.value)}
								onValueChange={(level) =>
									(input.value = withLevel(resource, input.value, level as ScopeLevel))}
							>
								{#each levelsOf(resource) as level (level.value)}
									<label class="text-control flex items-center gap-1.5 {level.column}">
										<RadioGroup.Item value={level.value} />
										{level.label}
									</label>
								{/each}
							</RadioGroup.Root>
						</li>
					{/each}
				</ul>
			</section>
		{/each}
	{/if}
	{#if input.error}
		<Field.Error>{input.error}</Field.Error>
	{/if}
</Field.Set>
