<script lang="ts">
	import DurationInput from '#lib/components/form/duration-input.svelte';
	import FormInput from '#lib/components/form/form-input.svelte';
	import SettingRow from '#lib/components/setting-row.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import { Switch } from '@pocket-id/ui/switch';
	import { m } from '#lib/paraglide/messages.js';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { AllAppConfig } from '#lib/types/application-configuration.type.ts';
	import { createForm, pickSchemaValues } from '#lib/utils/form-util.ts';
	import { trackFormChanges } from '#lib/utils/unsaved-changes-util.svelte.ts';
	import HouseIcon from '~icons/solar/home-2-bold-duotone';
	import UserPenIcon from '~icons/solar/pen-new-square-bold-duotone';
	import { z } from 'zod/v4';
	import OptionSelect from './option-select.svelte';

	let {
		callback,
		appConfig
	}: {
		appConfig: AllAppConfig;
		callback: (appConfig: Partial<AllAppConfig>) => Promise<void>;
	} = $props();

	const homePageUrlOptions = [
		{ label: m.my_account(), value: '/settings/account' },
		{ label: m.my_apps(), value: '/settings/apps' }
	];

	const formSchema = z.object({
		appName: z.string().min(2).max(30),
		homePageUrl: z.string(),
		sessionDuration: z
			.number()
			.min(1)
			.max(43200, { message: m.config_session_duration_maximum() })
			.refine((minutes) => Number.isInteger(minutes), {
				message: m.config_session_duration_whole_minutes()
			}),
		allowOwnAccountEdit: z.boolean()
	});

	const formStore = createForm(formSchema, pickSchemaValues(formSchema, appConfig));
	const inputs = formStore.inputs;

	trackFormChanges(() => formStore, callback);
</script>

<fieldset class="flex min-w-0 flex-col gap-10" disabled={$appConfigStore.uiConfigDisabled}>
	<SettingsSection title={m.application()}>
		<div class="flex flex-col gap-4 py-4 shadow-[inset_0_-1px_0_var(--border)]">
			<FormInput class="max-w-[400px]" label={m.application_name()} bind:input={$inputs.appName} />
			<div class="max-w-[400px]">
				<DurationInput
					id="session-duration"
					label={m.session_duration()}
					bind:input={$inputs.sessionDuration}
				/>
			</div>
		</div>
		<SettingRow
			title={m.app_config_home_page()}
			description={m.config_home_page_description()}
			icon={HouseIcon}
		>
			{#snippet control(props)}
				<OptionSelect
					{...props}
					value={$inputs.homePageUrl.value}
					options={homePageUrlOptions}
					onValueChange={(v) => ($inputs.homePageUrl.value = v)}
				/>
			{/snippet}
		</SettingRow>
		<SettingRow
			title={m.config_self_account_editing()}
			description={m.config_self_account_editing_description()}
			icon={UserPenIcon}
		>
			{#snippet control(props)}
				<Switch {...props} bind:checked={$inputs.allowOwnAccountEdit.value} />
			{/snippet}
		</SettingRow>
	</SettingsSection>
</fieldset>
