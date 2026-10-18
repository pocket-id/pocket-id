<script lang="ts">
	import CustomClaimsInput from '#lib/components/form/custom-claims-input.svelte';
	import UserGroupInput from '#lib/components/form/user-group-input.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import * as Field from '#lib/components/ui/field/index.ts';
	import * as RadioGroup from '#lib/components/ui/radio-group/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { AllAppConfig } from '#lib/types/application-configuration.type.ts';
	import { trackUnsavedValue } from '#lib/utils/unsaved-changes-util.svelte.ts';
	import { LucideInfo } from '@lucide/svelte';

	let {
		appConfig,
		showCustomClaims,
		callback
	}: {
		appConfig: AllAppConfig;
		showCustomClaims: boolean;
		callback: (updatedConfig: Partial<AllAppConfig>) => Promise<void>;
	} = $props();

	let selectedGroupIds = $state<string[]>(appConfig.signupDefaultUserGroupIDs ?? []);
	let customClaims = $state(appConfig.signupDefaultCustomClaims ?? []);
	let allowUserSignups = $state(appConfig.allowUserSignups);

	const appName = $derived($appConfigStore.appName);
	const signupOptions: { value: string; label: string; description?: string }[] = $derived([
		{ value: 'disabled', label: m.config_signup_disabled() },
		{
			value: 'withToken',
			label: m.config_signup_with_token(),
			description: m.config_signup_with_token_description()
		},
		{
			value: 'open',
			label: m.config_signup_open(),
			description: m.config_signup_open_description({ appName })
		}
	]);

	trackUnsavedValue(
		() => ({
			allowUserSignups,
			signupDefaultUserGroupIDs: selectedGroupIds,
			signupDefaultCustomClaims: customClaims
		}),
		(value) => {
			allowUserSignups = value.allowUserSignups;
			selectedGroupIds = value.signupDefaultUserGroupIDs;
			customClaims = value.signupDefaultCustomClaims;
		},
		callback
	);
</script>

<fieldset class="flex min-w-0 flex-col gap-10" disabled={$appConfigStore.uiConfigDisabled}>
	<SettingsSection
		title={m.config_who_can_sign_up()}
		description={m.config_who_can_sign_up_description()}
	>
		<div class="flex flex-col gap-3 pt-2">
			<RadioGroup.Root
				bind:value={
					() => allowUserSignups, (v) => (allowUserSignups = v as AllAppConfig['allowUserSignups'])
				}
				aria-label={m.config_who_can_sign_up()}
				class="gap-2"
			>
				{#each signupOptions as option (option.value)}
					<Field.Label for="signup-{option.value}">
						<Field.Field orientation="horizontal" class="items-start">
							<RadioGroup.Item id="signup-{option.value}" value={option.value} class="mt-0.5" />
							<Field.Content>
								<Field.Title>{option.label}</Field.Title>
								{#if option.description}
									<Field.Description>{option.description}</Field.Description>
								{/if}
							</Field.Content>
						</Field.Field>
					</Field.Label>
				{/each}
			</RadioGroup.Root>
			{#if allowUserSignups === 'withToken'}
				<p class="text-caption text-fg-muted flex items-start gap-1.5" aria-live="polite">
					<LucideInfo class="mt-[3px] size-3 shrink-0" />
					<a href="/settings/admin/users" class="inline-link">{m.config_manage_signup_tokens()}</a>
				</p>
			{/if}
		</div>
	</SettingsSection>

	<SettingsSection title={m.config_new_accounts()}>
		<div class="flex flex-col gap-6 pt-2">
			<Field.Field class="max-w-[400px]">
				<Field.Label for="default-groups">{m.config_default_groups()}</Field.Label>
				<UserGroupInput bind:selectedGroupIds />
				<Field.Description>{m.config_default_groups_description()}</Field.Description>
			</Field.Field>
			{#if showCustomClaims}
				<Field.Field>
					<Field.Label>{m.config_default_custom_claims()}</Field.Label>
					<CustomClaimsInput bind:customClaims />
				</Field.Field>
			{/if}
		</div>
	</SettingsSection>
</fieldset>
