<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import Label from '$lib/components/ui/label/label.svelte';
	import * as Select from '$lib/components/ui/select';
	import { m } from '$lib/paraglide/messages';
	import type {
		OidcClient,
		OidcClientClaimRemapping,
		ClaimRemappingSourceType
	} from '$lib/types/oidc.type';
	import { LucideMinus, LucidePlus } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';
	import { z } from 'zod/v4';

	let {
		client,
		claimRemappings = $bindable([]),
		errors,
		...restProps
	}: HTMLAttributes<HTMLDivElement> & {
		client?: OidcClient;
		claimRemappings: OidcClientClaimRemapping[];
		errors?: z.core.$ZodIssue[];
		children?: Snippet;
	} = $props();

	// User fields that can be used as sources
	const userFields = [
		{ value: 'email', label: 'Email' },
		{ value: 'first_name', label: 'First Name' },
		{ value: 'last_name', label: 'Last Name' },
		{ value: 'display_name', label: 'Display Name' },
		{ value: 'username', label: 'Username' },
		{ value: 'locale', label: 'Locale' }
	];

	const sourceTypes = [
		{ value: 'user_field', label: 'User Field' },
		{ value: 'custom_claim', label: 'Custom Claim' },
		{ value: 'static', label: 'Static Value' }
	];

	function addClaimRemapping() {
		claimRemappings = [
			...claimRemappings,
			{
				claimName: '',
				sourceType: 'user_field' as ClaimRemappingSourceType,
				sourceValue: ''
			}
		];
	}

	function removeClaimRemapping(index: number) {
		claimRemappings = claimRemappings.filter((_, i) => i !== index);
	}

	function updateClaimRemapping(
		index: number,
		field: keyof OidcClientClaimRemapping,
		value: string
	) {
		claimRemappings[index] = {
			...claimRemappings[index],
			[field]: value
		};
	}

	function getFieldError(index: number, field: keyof OidcClientClaimRemapping): string | null {
		if (!errors) return null;
		const path = [index, field];
		return errors?.filter((e) => e.path[0] == path[0] && e.path[1] == path[1])[0]?.message;
	}
</script>

<div {...restProps}>
	<FormInput
		label="Claim Remappings"
		description="Remap standard claims to different sources for this client. For example, map 'email' to a custom claim value to return a user's work email instead of their Pocket-ID email."
		docsLink="https://pocket-id.org/docs/guides/claim-remapping"
	>
		<div class="space-y-4">
			{#each claimRemappings as remapping, i}
				<div class="space-y-3 rounded-lg border p-4">
					<div class="flex items-center justify-between">
						<Label class="text-sm font-medium">Remapping {i + 1}</Label>
						<Button
							variant="outline"
							size="sm"
							onclick={() => removeClaimRemapping(i)}
							aria-label="Remove claim remapping"
						>
							<LucideMinus class="size-4" />
						</Button>
					</div>

					<div class="grid grid-cols-1 gap-3 md:grid-cols-3">
						<!-- Claim Name -->
						<div>
							<Label required for="claim-name-{i}" class="text-xs">Claim Name</Label>
							<Input
								id="claim-name-{i}"
								placeholder="email"
								value={remapping.claimName}
								oninput={(e) => updateClaimRemapping(i, 'claimName', e.currentTarget.value)}
								aria-invalid={!!getFieldError(i, 'claimName')}
							/>
							{#if getFieldError(i, 'claimName')}
								<p class="text-destructive mt-1 text-xs">{getFieldError(i, 'claimName')}</p>
							{/if}
						</div>

						<!-- Source Type -->
						<div>
							<Label required for="source-type-{i}" class="text-xs">Source Type</Label>
							<Select.Root
								selected={{
									value: remapping.sourceType,
									label: sourceTypes.find((st) => st.value === remapping.sourceType)?.label ||
										remapping.sourceType
								}}
								onSelectedChange={(selected) => {
									if (selected) {
										updateClaimRemapping(i, 'sourceType', selected.value);
										// Clear sourceValue when changing type
										updateClaimRemapping(i, 'sourceValue', '');
									}
								}}
							>
								<Select.Trigger id="source-type-{i}">
									<Select.Value />
								</Select.Trigger>
								<Select.Content>
									{#each sourceTypes as sourceType}
										<Select.Item value={sourceType.value}>{sourceType.label}</Select.Item>
									{/each}
								</Select.Content>
							</Select.Root>
						</div>

						<!-- Source Value -->
						<div>
							<Label required for="source-value-{i}" class="text-xs">Source Value</Label>
							{#if remapping.sourceType === 'user_field'}
								<Select.Root
									selected={{
										value: remapping.sourceValue,
										label: userFields.find((uf) => uf.value === remapping.sourceValue)?.label ||
											remapping.sourceValue
									}}
									onSelectedChange={(selected) => {
										if (selected) updateClaimRemapping(i, 'sourceValue', selected.value);
									}}
								>
									<Select.Trigger id="source-value-{i}">
										<Select.Value />
									</Select.Trigger>
									<Select.Content>
										{#each userFields as field}
											<Select.Item value={field.value}>{field.label}</Select.Item>
										{/each}
									</Select.Content>
								</Select.Root>
							{:else if remapping.sourceType === 'custom_claim'}
								<Input
									id="source-value-{i}"
									placeholder="company_email"
									value={remapping.sourceValue}
									oninput={(e) => updateClaimRemapping(i, 'sourceValue', e.currentTarget.value)}
									aria-invalid={!!getFieldError(i, 'sourceValue')}
								/>
							{:else}
								<Input
									id="source-value-{i}"
									placeholder="Static value or JSON"
									value={remapping.sourceValue}
									oninput={(e) => updateClaimRemapping(i, 'sourceValue', e.currentTarget.value)}
									aria-invalid={!!getFieldError(i, 'sourceValue')}
								/>
							{/if}
							{#if getFieldError(i, 'sourceValue')}
								<p class="text-destructive mt-1 text-xs">{getFieldError(i, 'sourceValue')}</p>
							{/if}
						</div>
					</div>
				</div>
			{/each}
		</div>
	</FormInput>

	<Button class="mt-3" variant="secondary" size="sm" onclick={addClaimRemapping} type="button">
		<LucidePlus class="mr-1 size-4" />
		{claimRemappings.length === 0 ? 'Add Claim Remapping' : 'Add Another Claim Remapping'}
	</Button>
</div>
