<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Field from '$lib/components/ui/field';
	import { Input } from '$lib/components/ui/input';
	import * as Select from '$lib/components/ui/select';
	import { m } from '$lib/paraglide/messages';
	import type { OidcClientClaimMapping } from '$lib/types/oidc.type';
	import { LucideMinus } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';
	import { z } from 'zod/v4';

	let {
		idx,
		claimMapping = $bindable({
			claimName: '',
			sourceType: 'user_field',
			sourceValue: '',
			conflictStrategy: 'default'
		}),
		onRemove,
		errors,
		...restProps
	}: HTMLAttributes<HTMLDivElement> & {
		idx: number;
		claimMapping: OidcClientClaimMapping;
		onRemove: (idx: number) => void;
		errors?: z.core.$ZodIssue[];
		children?: Snippet;
	} = $props();

	let selectedSourceType = $state(claimMapping.sourceType);
	let selectedConflictStrategy = $state(claimMapping.conflictStrategy)
	let selectedUserAttribute = $state(claimMapping.sourceValue);

	// User fields that can be used as sources
	const userFields = [
		{ value: 'email', label: m.email() },
		{ value: 'first_name', label: m.first_name() },
		{ value: 'last_name', label: m.last_name() },
		{ value: 'display_name', label: m.display_name() },
		{ value: 'username', label: m.username() },
		{ value: 'locale', label: m.locale() }
	];

	const sourceTypes = [
		{ value: 'user_field', label: m.user_attribute() },
		{ value: 'custom_claim', label: m.custom_claim() },
		{ value: 'static', label: m.static_value() }
	];

	const conflictStrategies = [
		{ value: 'default', label: m.conflict_strategy_default() },
		{ value: 'first', label: m.conflict_strategy_first() },
		{ value: 'last', label: m.conflict_strategy_last() },
		{ value: 'collect', label: m.conflict_strategy_collect() }
	];

	function getFieldError(idx: number, field: keyof OidcClientClaimMapping): string | null {
		if (!errors) return null;
		return errors?.filter((e) => e.path[0] === idx && e.path[1] === field)[0]?.message;
	}
</script>

<div {...restProps}>
	<div class="space-y-3 rounded-lg border p-4">
		<div class="flex items-center justify-between">
			<Field.Label>Mapping {idx}</Field.Label>
			<Button
				variant="outline"
				size="sm"
				onclick={() => onRemove(idx)}
				aria-label={m.remove_claim_mapping()}
			>
				<LucideMinus class="size-4" />
			</Button>
		</div>

		<div class="grid grid-cols-1 gap-3 md:grid-cols-3">
			<!-- Claim Name -->
			<Field.Field>
				<Field.Label required for="claim-name-{idx}">{m.claim_name()}</Field.Label>
				<Input
					id="claim-name-{idx}"
					placeholder="preferred_username"
					value={claimMapping.claimName}
					oninput={(e) => {
						claimMapping.claimName = e.currentTarget.value;
					}}
					aria-invalid={!!getFieldError(idx, 'claimName')}
				/>
				{#if getFieldError(idx, 'claimName')}
					<Field.Error>{getFieldError(idx, 'claimName')}</Field.Error>
				{/if}
			</Field.Field>

			<!-- Source Type -->
			<Field.Field>
				<Field.Label required for="source-type-{idx}">{m.claim_source()}</Field.Label>
				<Select.Root
					type="single"
					value={selectedSourceType}
					onValueChange={(selected) => {
						// Clear sourceValue when changing type
						selectedSourceType = selected as typeof selectedSourceType;
						claimMapping.sourceType = selected as OidcClientClaimMapping['sourceType'];
						selectedUserAttribute = '';
					}}
				>
					<Select.Trigger id="source-type-{idx}">
						{sourceTypes.find((st) => (st.value as string) === (selectedSourceType as string))
							?.label}
					</Select.Trigger>
					<Select.Content>
						{#each sourceTypes as sourceType (sourceType.value)}
							<Select.Item value={sourceType.value}>{sourceType.label}</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			</Field.Field>

			<!-- Source Value -->
			<Field.Field>
				{#if selectedSourceType === 'user_field'}
					<Field.Label required for="source-value-{idx}">{m.claim_value()}</Field.Label>
					<Select.Root
						type="single"
						value={selectedUserAttribute}
						onValueChange={(selected) => {
							selectedUserAttribute = selected;
							claimMapping.sourceValue = selected;
						}}
					>
						<Select.Trigger id="source-value-{idx}">
							{userFields.find((uf) => uf.value === selectedUserAttribute)?.label ||
								selectedUserAttribute}
						</Select.Trigger>
						<Select.Content>
							{#each userFields as field (field.value)}
								<Select.Item value={field.value}>{field.label}</Select.Item>
							{/each}
						</Select.Content>
					</Select.Root>
				{:else if selectedSourceType === 'custom_claim'}
					<div class="flex items-start gap-2">
						<Field.Field class="flex-1">
							<Field.Label required for="source-value-{idx}">{m.claim_value()}</Field.Label>
							<Input
								id="source-value-{idx}"
								placeholder="company_email"
								value={selectedUserAttribute}
								oninput={(e) => {
									claimMapping.sourceValue = e.currentTarget.value;
									selectedUserAttribute = e.currentTarget.value;
								}}
								aria-invalid={!!getFieldError(idx, 'sourceValue')}
							/>
						</Field.Field>
						<Field.Field class="flex-0">
							<Field.Label for="conflict-strategy-{idx}">{m.conflict_strategy()}</Field.Label>
							<Select.Root
								type="single"
								value={selectedConflictStrategy}
								onValueChange={(selected) => {
									selectedConflictStrategy = selected;
									claimMapping.conflictStrategy = selected;
								}}
							>
								<Select.Trigger id="conflict-strategy-{idx}">
									{conflictStrategies.find((cs) => cs.value === selectedConflictStrategy)?.label || 'Default'}
								</Select.Trigger>
								<Select.Content>
									{#each conflictStrategies as strategy (strategy.value)}
										<Select.Item value={strategy.value}>{strategy.label}</Select.Item>
									{/each}
								</Select.Content>
							</Select.Root>
						</Field.Field>
					</div>
				{:else}
					<Field.Label required for="source-value-{idx}">{m.claim_value()}</Field.Label>
					<Input
						id="source-value-{idx}"
						placeholder="Static value"
						value={selectedUserAttribute}
						oninput={(e) => {
							claimMapping.sourceValue = e.currentTarget.value;
							selectedUserAttribute = e.currentTarget.value;
						}}
						aria-invalid={!!getFieldError(idx, 'sourceValue')}
					/>
				{/if}
				{#if getFieldError(idx, 'sourceValue')}
					<Field.Error>{getFieldError(idx, 'sourceValue')}</Field.Error>
				{/if}
			</Field.Field>
		</div>
	</div>
</div>
