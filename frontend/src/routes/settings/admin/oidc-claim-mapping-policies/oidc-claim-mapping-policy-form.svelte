<script lang="ts">
	import FormInput from '#lib/components/form/form-input.svelte';
	import SwitchWithLabel from '#lib/components/form/switch-with-label.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import type {
		OidcClaimMappingDto,
		OidcClaimMappingPolicy,
		OidcScope
	} from '#lib/types/oidc-claim-mapping-policy.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { preventDefault } from '#lib/utils/event-util.ts';
	import { createForm } from '#lib/utils/form-util.ts';
	import { trackFormChanges } from '#lib/utils/unsaved-changes-util.svelte.ts';
	import { z } from 'zod/v4';

	let {
		callback,
		existingPolicy,
		mode
	}: {
		existingPolicy?: OidcClaimMappingPolicy;
		callback: (policy: OidcClaimMappingPolicy) => Promise<void>;
		mode: 'create' | 'update';
	} = $props();

	let isLoading = $state(false);

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

	// Form validation schema for a single claim mapping
	const claimMappingSchema = z.object({
		claimName: z.string().min(1),
		sourceType: z.enum(['user_field', 'custom_claim', 'static']),
		sourceValue: z.string().min(1),
		scope: z.array(z.enum(availableScopes)),
		accessToken: z.boolean(),
		idToken: z.boolean(),
		userInfo: z.boolean()
	});

	// Form validation schema for the policy
	const policySchema = z.object({
		name: z.string().min(1),
		isDefault: z.boolean(),
		claimMappings: z.array(claimMappingSchema).min(1)
	});

	type FormSchema = typeof policySchema;
	// Claim mappings a new policy starts with, mirroring the standard OIDC claims
	const defaultClaimMappings: OidcClaimMappingDto[] = [
		{
			claimName: 'sub',
			sourceType: 'user_field',
			sourceValue: 'id',
			scope: ['openid'],
			accessToken: true,
			idToken: true,
			userInfo: true
		},
		{
			claimName: 'email',
			sourceType: 'user_field',
			sourceValue: 'email',
			scope: ['email'],
			accessToken: false,
			idToken: true,
			userInfo: true
		},
		{
			claimName: 'email_verified',
			sourceType: 'user_field',
			sourceValue: 'email_verified',
			scope: ['email'],
			accessToken: false,
			idToken: true,
			userInfo: true
		},
		{
			claimName: 'given_name',
			sourceType: 'user_field',
			sourceValue: 'first_name',
			scope: ['profile'],
			accessToken: false,
			idToken: true,
			userInfo: true
		},
		{
			claimName: 'family_name',
			sourceType: 'user_field',
			sourceValue: 'last_name',
			scope: ['profile'],
			accessToken: false,
			idToken: true,
			userInfo: true
		},
		{
			claimName: 'name',
			sourceType: 'user_field',
			sourceValue: 'full_name',
			scope: ['profile'],
			accessToken: false,
			idToken: true,
			userInfo: true
		},
		{
			claimName: 'display_name',
			sourceType: 'user_field',
			sourceValue: 'display_name',
			scope: ['profile'],
			accessToken: false,
			idToken: true,
			userInfo: true
		},
		{
			claimName: 'preferred_username',
			sourceType: 'user_field',
			sourceValue: 'username',
			scope: ['profile'],
			accessToken: false,
			idToken: true,
			userInfo: true
		},
		{
			claimName: 'locale',
			sourceType: 'user_field',
			sourceValue: 'locale',
			scope: ['profile'],
			accessToken: false,
			idToken: true,
			userInfo: true
		},
		{
			claimName: 'picture',
			sourceType: 'user_field',
			sourceValue: 'picture',
			scope: ['profile'],
			accessToken: false,
			idToken: true,
			userInfo: true
		},
		{
			claimName: 'groups',
			sourceType: 'user_field',
			sourceValue: 'groups',
			scope: ['groups'],
			accessToken: false,
			idToken: true,
			userInfo: true
		},
		{
			claimName: '*',
			sourceType: 'custom_claim',
			sourceValue: '*',
			scope: ['profile'],
			accessToken: false,
			idToken: true,
			userInfo: true
		}
	];

	// Initial form data

	const initialData = {
		name: existingPolicy?.name ?? '',
		isDefault: existingPolicy?.isDefault ?? false,
		claimMappings: existingPolicy?.claimMappings ?? defaultClaimMappings
	};

	const formStore = createForm<FormSchema>(policySchema, initialData);
	const { inputs } = formStore;

	// save policy
	async function savePolicy(data: z.infer<FormSchema>) {
		await callback({
			name: data.name,
			isDefault: data.isDefault,
			claimMappings: data.claimMappings
		});
		if (!existingPolicy) formStore.reset();
	}

	// Handle form submission
	async function onSubmit() {
		const data = formStore.validate();
		if (!data) return;
		isLoading = true;
		try {
			await savePolicy(data);
		} catch (e) {
			axiosErrorToast(e);
		} finally {
			isLoading = false;
		}
	}

	if (mode === 'update') {
		trackFormChanges(() => formStore, savePolicy);
	}
</script>

<form onsubmit={preventDefault(onSubmit)} class="flex flex-col gap-6">
	<!-- Policy name and default checkbox -->
	<div class="grid grid-cols-1 gap-x-3 gap-y-7 sm:flex-row md:grid-cols-2">
		<FormInput
			label={m.name()}
			class="w-full"
			bind:input={$inputs.name}
			placeholder={m.enter_policy_name()}
		/>
		<div>
			<SwitchWithLabel
				id="isDefault"
				label={m.set_as_default()}
				description={m.set_as_default_description()}
				onCheckedChange={(v) => {
					if (v) {
						$inputs.isDefault.value = true;
					}
				}}
				bind:checked={$inputs.isDefault.value}
			/>
		</div>
	</div>

	<!-- Form actions -->
	{#if mode === 'create'}
		<div class="relative mt-5 flex justify-end">
			<Button {isLoading} type="submit">{m.save()}</Button>
		</div>
	{/if}
</form>
