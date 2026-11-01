<script lang="ts">
	import * as Field from '#lib/components/ui/field/index.ts';
	import * as RadioGroup from '#lib/components/ui/radio-group/index.ts';
	import { m } from '#lib/paraglide/messages.js';

	let {
		isPublic = $bindable(),
		disabled = false
	}: {
		isPublic: boolean;
		disabled?: boolean;
	} = $props();

	const options = [
		{
			value: 'confidential',
			title: m.confidential(),
			description: m.confidential_client_tile_description()
		},
		{ value: 'public', title: m.public(), description: m.public_client_type_description() }
	];
</script>

<Field.Set>
	<Field.Legend variant="label">{m.client_type()}</Field.Legend>
	<RadioGroup.Root
		class="grid gap-3 sm:grid-cols-2"
		{disabled}
		bind:value={
			() => (isPublic ? 'public' : 'confidential'), (value) => (isPublic = value === 'public')
		}
	>
		{#each options as option (option.value)}
			<Field.Label for="client-type-{option.value}">
				<Field.Field orientation="horizontal" data-disabled={disabled}>
					<RadioGroup.Item value={option.value} id="client-type-{option.value}" />
					<Field.Content>
						<Field.Title>{option.title}</Field.Title>
						<Field.Description>{option.description}</Field.Description>
					</Field.Content>
				</Field.Field>
			</Field.Label>
		{/each}
	</RadioGroup.Root>
</Field.Set>
