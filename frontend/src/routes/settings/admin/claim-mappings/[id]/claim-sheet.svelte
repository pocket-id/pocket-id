<script lang="ts">
	import AutoCompleteInput from '#lib/components/form/auto-complete-input.svelte';
	import FormInput from '#lib/components/form/form-input.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { Checkbox } from '#lib/components/ui/checkbox/index.ts';
	import * as Field from '#lib/components/ui/field/index.ts';
	import * as Select from '#lib/components/ui/select/index.ts';
	import * as Sheet from '#lib/components/ui/sheet/index.ts';
	import * as Tabs from '#lib/components/ui/tabs/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import CustomClaimService from '#lib/services/custom-claim-service.ts';
	import {
		MAPPING_SCOPES,
		RESERVED_CLAIM_NAMES,
		USER_FIELDS,
		type OidcClaimMappingDto,
		type OidcClaimMappingPolicySourceType,
		type OidcScope,
		type OidcUserField
	} from '#lib/types/oidc-claim-mapping-policy.type.ts';
	import { preventDefault } from '#lib/utils/event-util.ts';
	import { LucideTrash2 } from '@lucide/svelte';
	import { untrack } from 'svelte';
	import { userFieldLabel } from '../claim-mapping-util';

	let {
		open = $bindable(),
		mapping,
		others,
		onSave,
		onRemove
	}: {
		open: boolean;
		// The claim being edited, or null to add one
		mapping: OidcClaimMappingDto | null;
		// The policy's other claims, which a new or renamed claim must not collide with
		others: OidcClaimMappingDto[];
		onSave: (mapping: OidcClaimMappingDto) => void;
		onRemove: () => void;
	} = $props();

	const TOKENS = [
		{ key: 'idToken', label: m.id_token },
		{ key: 'accessToken', label: m.access_token },
		{ key: 'userInfo', label: m.userinfo_response }
	] as const;

	const formId = $props.id();
	const customClaimService = new CustomClaimService();

	let claim = $state({ value: '', error: null as string | null });
	let sourceType = $state<OidcClaimMappingPolicySourceType>('user_field');
	let userField = $state<OidcUserField>('email');
	let customKey = $state({ value: '', error: null as string | null });
	let allCustom = $state(false);
	let staticValue = $state({ value: '', error: null as string | null });
	let scopes = $state<OidcScope[]>([]);
	let tokens = $state({ idToken: false, accessToken: false, userInfo: false });
	let scopesError = $state<string | null>(null);
	let tokensError = $state<string | null>(null);
	let suggestions = $state<string[]>([]);

	// Every opening starts from the claim as it is, or from what most apps read for a new one
	$effect(() => {
		if (!open) return;
		untrack(() => {
			const base = mapping ?? {
				claimName: '',
				sourceType: 'user_field',
				sourceValue: 'email',
				scope: ['profile'],
				idToken: true,
				accessToken: false,
				userInfo: true
			};
			const wildcard = base.sourceType === 'custom_claim' && base.sourceValue === '*';
			claim = { value: wildcard ? '' : base.claimName, error: null };
			sourceType = base.sourceType;
			userField = base.sourceType === 'user_field' ? (base.sourceValue as OidcUserField) : 'email';
			customKey = {
				value: base.sourceType === 'custom_claim' && !wildcard ? base.sourceValue : '',
				error: null
			};
			allCustom = wildcard;
			staticValue = { value: base.sourceType === 'static' ? base.sourceValue : '', error: null };
			scopes = [...base.scope];
			tokens = { idToken: base.idToken, accessToken: base.accessToken, userInfo: base.userInfo };
			scopesError = null;
			tokensError = null;
			customClaimService.getSuggestions().then((keys) => (suggestions = keys));
		});
	});

	const wildcard = $derived(sourceType === 'custom_claim' && allCustom);

	function claimError(name: string) {
		if (wildcard) {
			return others.some((o) => o.claimName === '*')
				? m.policy_already_passes_every_custom_claim()
				: null;
		}
		if (!name) return m.enter_the_claim_name_apps_read();
		if (RESERVED_CLAIM_NAMES.includes(name)) return m.claim_is_set_by_pocket_id({ name });
		if (/\s/.test(name)) return m.claim_names_cant_contain_spaces();
		if (others.some((o) => o.claimName === name)) return m.policy_already_has_claim({ name });
		return null;
	}

	function submit() {
		const name = claim.value.trim();
		claim.error = claimError(name);
		customKey.error =
			sourceType === 'custom_claim' && !allCustom && !customKey.value.trim()
				? m.enter_a_custom_claim_key()
				: null;
		staticValue.error =
			sourceType === 'static' && !staticValue.value ? m.enter_the_value_every_person_gets() : null;
		scopesError = scopes.length ? null : m.pick_at_least_one_scope();
		tokensError = Object.values(tokens).some(Boolean)
			? null
			: m.pick_at_least_one_place_to_include_it();
		if (claim.error || customKey.error || staticValue.error || scopesError || tokensError) return;

		const sourceValue =
			sourceType === 'user_field'
				? userField
				: sourceType === 'custom_claim'
					? allCustom
						? '*'
						: customKey.value.trim()
					: staticValue.value;
		onSave({
			claimName: wildcard ? '*' : name,
			sourceType,
			sourceValue,
			scope: MAPPING_SCOPES.filter((s) => scopes.includes(s)),
			...tokens
		});
		open = false;
	}

	function toggleScope(scope: OidcScope, checked: boolean) {
		scopes = checked ? [...scopes, scope] : scopes.filter((s) => s !== scope);
		scopesError = null;
	}
</script>

<Sheet.Root bind:open>
	<Sheet.Content title={mapping ? m.edit_claim() : m.add_claim()}>
		<form id={formId} onsubmit={preventDefault(submit)} class="flex flex-col gap-6">
			<FormInput
				label={m.claim_name()}
				description={wildcard ? m.custom_claims_keep_their_own_key() : m.claim_name_helper()}
				placeholder="roles"
				autocomplete="off"
				inputClass="font-mono"
				disabled={wildcard}
				bind:input={claim}
			/>

			<Field.Field>
				<Field.Label>{m.value_from()}</Field.Label>
				<Tabs.Root bind:value={sourceType}>
					<Tabs.List aria-label={m.value_from()}>
						<Tabs.Trigger value="user_field">{m.user_field()}</Tabs.Trigger>
						<Tabs.Trigger value="custom_claim">{m.custom_claim()}</Tabs.Trigger>
						<Tabs.Trigger value="static">{m.fixed_value()}</Tabs.Trigger>
					</Tabs.List>
				</Tabs.Root>
			</Field.Field>

			{#if sourceType === 'user_field'}
				<Field.Field>
					<Field.Label for="claim-user-field">{m.user_field()}</Field.Label>
					<Select.Root type="single" bind:value={userField}>
						<Select.Trigger id="claim-user-field" class="w-full">
							{userFieldLabel(userField)}
						</Select.Trigger>
						<Select.Content>
							{#each USER_FIELDS as field (field)}
								<Select.Item value={field} label={userFieldLabel(field)} />
							{/each}
						</Select.Content>
					</Select.Root>
				</Field.Field>
			{:else if sourceType === 'custom_claim'}
				<div class="flex flex-col gap-3">
					{#if !allCustom}
						<FormInput
							label={m.custom_claim_key()}
							description={m.custom_claim_key_helper()}
							labelFor="claim-custom-key"
							input={customKey}
						>
							<AutoCompleteInput
								id="claim-custom-key"
								placeholder="department"
								{suggestions}
								bind:value={
									() => customKey.value,
									(value) => {
										customKey.value = value;
										customKey.error = null;
									}
								}
							/>
						</FormInput>
					{/if}
					<label class="flex items-start gap-3">
						<Checkbox
							class="mt-px"
							bind:checked={allCustom}
							onCheckedChange={() => (claim.error = null)}
						/>
						<span class="flex flex-col">
							<span class="text-control text-fg font-medium">
								{m.every_custom_claim_under_its_own_key()}
							</span>
							<span class="text-caption text-fg-muted">
								{m.every_custom_claim_under_its_own_key_description()}
							</span>
						</span>
					</label>
				</div>
			{:else}
				<FormInput
					label={m.fixed_value()}
					description={m.fixed_value_helper()}
					placeholder="acme"
					autocomplete="off"
					bind:input={staticValue}
				/>
			{/if}

			<Field.Set>
				<Field.Legend variant="label" class="mb-0">{m.released_with()}</Field.Legend>
				{#if scopesError}
					<Field.Error>{scopesError}</Field.Error>
				{:else}
					<Field.Description>{m.released_with_helper()}</Field.Description>
				{/if}
				<div class="grid grid-cols-2 gap-x-4 gap-y-2.5">
					{#each MAPPING_SCOPES as scope (scope)}
						<label class="text-mono-s text-fg flex items-center gap-3 font-mono">
							<Checkbox
								checked={scopes.includes(scope)}
								onCheckedChange={(checked) => toggleScope(scope, checked)}
								aria-invalid={!!scopesError}
							/>
							{scope}
						</label>
					{/each}
				</div>
			</Field.Set>

			<Field.Set>
				<Field.Legend variant="label" class="mb-0">{m.included_in()}</Field.Legend>
				{#if tokensError}
					<Field.Error>{tokensError}</Field.Error>
				{:else}
					<Field.Description>{m.included_in_helper()}</Field.Description>
				{/if}
				<div class="flex flex-col gap-2.5">
					{#each TOKENS as { key, label } (key)}
						<label class="text-control text-fg flex items-center gap-3">
							<Checkbox
								bind:checked={tokens[key]}
								onCheckedChange={() => (tokensError = null)}
								aria-invalid={!!tokensError}
							/>
							{label()}
						</label>
					{/each}
				</div>
			</Field.Set>
		</form>
		{#snippet footer()}
			{#if mapping}
				<Button
					variant="ghost"
					class="me-auto"
					onclick={() => {
						onRemove();
						open = false;
					}}
				>
					<LucideTrash2 />
					{m.remove_claim()}
				</Button>
			{/if}
			<Button variant="ghost" onclick={() => (open = false)}>{m.cancel()}</Button>
			<Button type="submit" form={formId}>{mapping ? m.done() : m.add_claim()}</Button>
		{/snippet}
	</Sheet.Content>
</Sheet.Root>
