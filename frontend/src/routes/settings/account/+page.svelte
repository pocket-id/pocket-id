<script lang="ts">
	import EmailVerificationStateBox from '#lib/components/email-verification-state-box.svelte';
	import FormattedMessage from '#lib/components/formatted-message.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import SettingRow from '#lib/components/setting-row.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import * as Alert from '#lib/components/ui/alert/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { LucideLanguages, LucideTriangleAlert } from '@lucide/svelte';
	import AccountForm from './account-form.svelte';
	import LocalePicker from './locale-picker.svelte';
	import PasskeyList from './passkey-list.svelte';

	let { data } = $props();

	let passkeys = $state(data.passkeys);
</script>

<svelte:head>
	<title>{m.my_account()}</title>
</svelte:head>

<div class="flex flex-col gap-6">
	<PageHeader title={m.my_account()} description={m.account_page_description()} />
	<EmailVerificationStateBox />
	{#if passkeys.length === 1}
		<Alert.Root variant="warning">
			<LucideTriangleAlert />
			<Alert.Title>{m.add_backup_passkey()}</Alert.Title>
			<Alert.Description class="wrap-anywhere">
				{m.add_backup_passkey_description({ name: passkeys[0].name })}
			</Alert.Description>
		</Alert.Root>
	{/if}
</div>

<div class="mt-6 flex flex-col gap-10">
	<AccountForm account={data.account} pendingEmailChange={data.pendingEmailChange} />

	<PasskeyList bind:passkeys />

	<SettingsSection title={m.preferences()}>
		<SettingRow title={m.language()} icon={LucideLanguages}>
			{#snippet description()}
				<FormattedMessage message={m.language_crowdin_description} />
			{/snippet}
			{#snippet control(props)}
				<LocalePicker {...props} />
			{/snippet}
		</SettingRow>
	</SettingsSection>
</div>
