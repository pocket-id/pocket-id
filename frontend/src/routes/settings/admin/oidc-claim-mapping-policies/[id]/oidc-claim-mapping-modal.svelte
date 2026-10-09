<script lang="ts">
	import FormInput from '#lib/components/form/form-input.svelte';
	import MultiSelect from '#lib/components/form/multi-select.svelte';
	import SwitchWithLabel from '#lib/components/form/switch-with-label.svelte';
	import * as Dialog from '#lib/components/ui/dialog/index.ts';
	import Input from '#lib/components/ui/input/input.svelte';
	import * as Select from '#lib/components/ui/select/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { RESERVED_CLAIM_NAMES } from '#lib/types/oidc-claim-mapping-policy.type.ts';
	import type {
		OidcClaimMappingDto,
		OidcClaimMappingPolicySourceType,
		OidcScope,
		OidcUserField
	} from '#lib/types/oidc-claim-mapping-policy.type.ts';
	import { preventDefault } from '#lib/utils/event-util.ts';
	import { createForm } from '#lib/utils/form-util.ts';
	import { z } from 'zod/v4';
	import Button from '#lib/components/ui/button/button.svelte';
	import AutoCompleteInput from '#lib/components/form/auto-complete-input.svelte';
	import CustomClaimService from '#lib/services/custom-claim-service.ts';
	import { onMount } from 'svelte';

	let {
		open = $bindable(),
		mapping,
		index,
		onSave
	}: {
		open: boolean;
		mapping: OidcClaimMappingDto;
		index: number;
		onSave: (mapping: OidcClaimMappingDto, index: number) => void;
	} = $props();

	// Suggested claim names
	const claimNameSuggestions: string[] = [
		'given_name',
		'family_name',
		'display_name',
		'name',
		'preferred_username',
		'locale',
		'email',
		'email_verified',
		'groups',
		'picture'
	];

	// Available user fields for source type 'user_field'
	const userFields: OidcUserField[] = [
		'id',
		'email',
		'email_verified',
		'first_name',
		'last_name',
		'full_name',
		'display_name',
		'username',
		'locale',
		'picture',
		'groups'
	];

	// Available scopes
	const availableScopes: OidcScope[] = [
		'openid',
		'profile',
		'email',
		'address',
		'phone',
		'offline_access',
		'groups'
	];

	// Available source types
	const sourceTypes = ['user_field', 'custom_claim', 'static'] as const;

	// Get user field source for display
	function getSourceTypeLabel(field: OidcClaimMappingPolicySourceType): string {
		const labels: Record<OidcClaimMappingPolicySourceType, string> = {
			user_field: m.user_field(),
			custom_claim: m.custom_claims(),
			static: m.static_value()
		};
		return labels[field] || field;
	}

	// Get user field label for display
	function getUserFieldLabel(field: OidcUserField): string {
		const labels: Record<OidcUserField, string> = {
			id: m.user_identifier(),
			email: m.email(),
			email_verified: m.email_verified(),
			first_name: m.first_name(),
			last_name: m.last_name(),
			full_name: m.full_name(),
			display_name: m.display_name(),
			username: m.username(),
			locale: m.locale(),
			picture: m.profile_picture(),
			groups: m.groups()
		};
		return labels[field] || field;
	}

	// Form validation schema for a single claim mapping
	const claimMappingSchema = z.object({
		claimName: z
			.string()
			.min(1)
			.refine(
				(value) => !RESERVED_CLAIM_NAMES.includes(value as (typeof RESERVED_CLAIM_NAMES)[number]),
				{
					message: m.this_claim_name_is_reserved_and_cannot_be_remapped()
				}
			),
		sourceType: z.enum(['user_field', 'custom_claim', 'static']),
		sourceValue: z.string().min(1),
		scope: z.array(z.enum(availableScopes)).min(1),
		accessToken: z.boolean(),
		idToken: z.boolean(),
		userInfo: z.boolean()
	});
	type FormSchema = typeof claimMappingSchema;
	const { inputs, ...form } = createForm<FormSchema>(claimMappingSchema, mapping);

	// Handle form submission
	function onSubmit() {
		const data = form.validate();
		if (!data) return;

		onSave(data, index);
		open = false;
	}

	// custom claim keys suggestions
	const customClaimService = new CustomClaimService();

	let customClaimKeysSuggestions: string[] = $state([]);
	onMount(() => {
		customClaimService.getSuggestions().then((data) => (customClaimKeysSuggestions = data));
	});
</script>

<Dialog.Root bind:open>
	<Dialog.Content class="max-h-[90vh] min-w-[90vw] overflow-auto lg:min-w-250">
		<Dialog.Header>
			<Dialog.Title>{m.claim_mapping()}</Dialog.Title>
			<Dialog.Description>{m.add_claim_mapping_description()}</Dialog.Description>
		</Dialog.Header>

		<form onsubmit={preventDefault(onSubmit)} class="flex flex-col gap-6">
			<div class="grid grid-cols-1 gap-x-3 gap-y-7 sm:flex-row md:grid-cols-2">
				<div class="flex flex-col gap-6">
					<!-- Claim Name -->
					<FormInput
						label={m.claim_name()}
						bind:input={$inputs.claimName}
						disabled={$inputs.claimName.value === 'sub'}
					>
						<AutoCompleteInput
							placeholder={m.enter_claim_name()}
							suggestions={claimNameSuggestions}
							bind:value={$inputs.claimName.value}
							disabled={$inputs.claimName.value === 'sub'}
						/>
					</FormInput>
					<!-- Source Type -->
					<FormInput label={m.source_type()} bind:input={$inputs.sourceType}>
						<Select.Root
							type="single"
							bind:value={$inputs.sourceType.value}
							onValueChange={() => {
								$inputs.sourceValue.value = '';
							}}
						>
							<Select.Trigger
								class="w-full sm:w-48"
								aria-label={m.source_type()}
								placeholder={m.source_type()}
							>
								{getSourceTypeLabel($inputs.sourceType.value)}
							</Select.Trigger>
							<Select.Content>
								{#each sourceTypes as option}
									<Select.Item value={option}>{getSourceTypeLabel(option)}</Select.Item>
								{/each}
							</Select.Content>
						</Select.Root>
					</FormInput>

					<!-- Source Value (depends on source type) -->
					<FormInput label={m.source_value()} bind:input={$inputs.sourceValue}>
						{#if $inputs.sourceType.value === 'user_field'}
							<Select.Root type="single" bind:value={$inputs.sourceValue.value}>
								<Select.Trigger
									class="w-full sm:w-48"
									aria-label={m.source_value()}
									placeholder={m.source_value()}
								>
									{getUserFieldLabel($inputs.sourceValue.value as OidcUserField)}
								</Select.Trigger>
								<Select.Content>
									{#each userFields as option}
										<Select.Item value={option}>{getUserFieldLabel(option)}</Select.Item>
									{/each}
								</Select.Content>
							</Select.Root>
						{:else if $inputs.sourceType.value === 'custom_claim'}
							<AutoCompleteInput
								placeholder={m.enter_custom_claim_key()}
								suggestions={customClaimKeysSuggestions}
								bind:value={$inputs.sourceValue.value}
							/>
						{:else}
							<Input bind:value={$inputs.sourceValue.value} placeholder={m.enter_static_value()} />
						{/if}
					</FormInput>
				</div>
				<div class="flex flex-col gap-6">
					<!-- Scope (multi-select) -->
					<FormInput
						label={m.scope()}
						bind:input={$inputs.scope}
						class="w-auto"
						disabled={$inputs.claimName.value === 'sub'}
					>
						<MultiSelect
							items={availableScopes.map((scope) => ({ value: scope, label: scope }))}
							bind:selectedItems={$inputs.scope.value}
							disabled={$inputs.claimName.value === 'sub'}
						/>
					</FormInput>
					<SwitchWithLabel
						id="accessToken"
						label={m.access_token()}
						description={m.add_to_access_token_description()}
						bind:checked={$inputs.accessToken.value}
						disabled={$inputs.claimName.value === 'sub'}
					/>
					<SwitchWithLabel
						id="id_token"
						label={m.id_token()}
						description={m.add_to_id_token_description()}
						bind:checked={$inputs.idToken.value}
						disabled={$inputs.claimName.value === 'sub'}
					/>
					<SwitchWithLabel
						id="user_info"
						label={m.userinfo()}
						description={m.add_to_user_info_description()}
						bind:checked={$inputs.userInfo.value}
						disabled={$inputs.claimName.value === 'sub'}
					/>
				</div>
			</div>
			<!-- Form actions -->
			<div class="relative mt-5 flex justify-end">
				<Button type="submit">{m.save()}</Button>
			</div>
		</form>
	</Dialog.Content>
</Dialog.Root>
