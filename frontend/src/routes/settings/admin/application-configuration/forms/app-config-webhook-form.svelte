<script lang="ts">
	import { openConfirmDialog } from '$lib/components/confirm-dialog';
	import FormInput from '$lib/components/form/form-input.svelte';
	import SearchableMultiSelect from '$lib/components/form/searchable-multi-select.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Label } from '$lib/components/ui/label';
	import { m } from '$lib/paraglide/messages';
	import AppConfigService from '$lib/services/app-config-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { AllAppConfig } from '$lib/types/application-configuration.type';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod/v4';

	let {
		callback,
		appConfig
	}: {
		appConfig: AllAppConfig;
		callback: (appConfig: Partial<AllAppConfig>) => Promise<void>;
	} = $props();

	const appConfigService = new AppConfigService();

	let isSendingTestWebhook = $state(false);

	const formSchema = z.object({
		webhookUrl: z.url().or(z.literal('')).optional(),
		webhookEvents: z.string().optional()
	});

	let { inputs, ...form } = $derived(createForm(formSchema, appConfig));

	const auditLogEventsList = [
		'SIGN_IN',
		'TOKEN_SIGN_IN',
		'ACCOUNT_CREATED',
		'CLIENT_AUTHORIZATION',
		'NEW_CLIENT_AUTHORIZATION',
		'DEVICE_CODE_AUTHORIZATION',
		'NEW_DEVICE_CODE_AUTHORIZATION',
		'PASSKEY_ADDED',
		'PASSKEY_REMOVED'
	] as const;

	function getEventLabel(event: string) {
		switch (event) {
			case 'SIGN_IN': return m.webhook_event_SIGN_IN();
			case 'TOKEN_SIGN_IN': return m.webhook_event_TOKEN_SIGN_IN();
			case 'ACCOUNT_CREATED': return m.webhook_event_ACCOUNT_CREATED();
			case 'CLIENT_AUTHORIZATION': return m.webhook_event_CLIENT_AUTHORIZATION();
			case 'NEW_CLIENT_AUTHORIZATION': return m.webhook_event_NEW_CLIENT_AUTHORIZATION();
			case 'DEVICE_CODE_AUTHORIZATION': return m.webhook_event_DEVICE_CODE_AUTHORIZATION();
			case 'NEW_DEVICE_CODE_AUTHORIZATION': return m.webhook_event_NEW_DEVICE_CODE_AUTHORIZATION();
			case 'PASSKEY_ADDED': return m.webhook_event_PASSKEY_ADDED();
			case 'PASSKEY_REMOVED': return m.webhook_event_PASSKEY_REMOVED();
			default: return event;
		}
	}

	const auditLogItems = auditLogEventsList.map(event => ({ value: event, label: getEventLabel(event) }));
	
	let selectedEvents = $state(appConfig.webhookEvents ? appConfig.webhookEvents.split(',') : []);

	function onSelectEvents(selected: string[]) {
		selectedEvents = selected;
		$inputs.webhookEvents.value = selected.join(',');
	}

	async function onSubmit() {
		const data = form.validate();
		if (!data) return false;
		await callback(data);

		// Update the app config to don't display the unsaved changes warning
		Object.entries(data).forEach(([key, value]) => {
			// @ts-ignore
			appConfig[key] = value;
		});

		toast.success(m.webhook_configuration_updated_successfully());
		return true;
	}

	async function onTestWebhook() {
		// @ts-ignore
		const hasChanges = Object.keys($inputs).some((key) => $inputs[key].value !== appConfig[key]);

		if (hasChanges) {
			openConfirmDialog({
				title: m.save_changes_question(),
				message: m.save_changes_before_test_webhook(),
				confirm: {
					label: m.save_and_send(),
					action: async () => {
						const saved = await onSubmit();
						if (saved) {
							sendTestWebhook();
						}
					}
				}
			});
		} else {
			sendTestWebhook();
		}
	}

	async function sendTestWebhook() {
		isSendingTestWebhook = true;
		await appConfigService
			.sendTestWebhook()
			.then(() => toast.success(m.test_webhook_sent_successfully()))
			.catch(() => toast.error(m.failed_to_send_test_webhook()))
			.finally(() => (isSendingTestWebhook = false));
	}
</script>

<form onsubmit={preventDefault(onSubmit)}>
	<fieldset disabled={$appConfigStore.uiConfigDisabled}>
		<h4 class="mb-4 text-lg font-semibold">{m.webhook_configuration()}</h4>
		<div class="mt-4 grid grid-cols-1 items-start gap-5">
			<FormInput label={m.webhook_url()} bind:input={$inputs.webhookUrl} />
		</div>
		<div class="mt-5">
			<Label for="webhook-events-select">{m.webhook_events()}</Label>
			<p class="mt-1 mb-3 text-sm text-muted-foreground">
				{m.webhook_events_description()}
			</p>
			<div>
				<SearchableMultiSelect 
					id="webhook-events-select" 
					items={auditLogItems} 
					selectedItems={selectedEvents} 
					onSelect={onSelectEvents} 
				/>
			</div>
		</div>
	</fieldset>
	<div class="mt-8 flex flex-wrap justify-end gap-3">
		<Button isLoading={isSendingTestWebhook} variant="secondary" onclick={onTestWebhook}
			>{m.send_test_webhook()}</Button
		>
		<Button type="submit" disabled={$appConfigStore.uiConfigDisabled}>{m.save()}</Button>
	</div>
</form>
