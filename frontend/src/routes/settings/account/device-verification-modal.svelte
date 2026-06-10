<script lang="ts">
	import CodeInput from '$lib/components/code-input.svelte';
	import FormattedMessage from '$lib/components/formatted-message.svelte';
	import ScopeList from '$lib/components/scope-list.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import * as Dialog from '$lib/components/ui/dialog';
	import { m } from '$lib/paraglide/messages';
	import OIDCService from '$lib/services/oidc-service';
	import type { OidcDeviceCodeInfo } from '$lib/types/oidc.type';
	import { getAxiosErrorMessage } from '$lib/utils/error-util';

	let {
		show = $bindable()
	}: {
		show: boolean;
	} = $props();

	const oidcService = new OIDCService();

	type ModalState = 'enter' | 'authorize-scopes' | 'success' | 'error';

	let modalState: ModalState = $state('enter');
	let isLoading = $state(false);
	let userCode = $state('');
	let errorMessage: string | null = $state(null);
	let deviceInfo: OidcDeviceCodeInfo | undefined = $state();

	let codeComplete = $derived(userCode.replace(/[^a-zA-Z0-9]/g, '').length >= 8);

	async function submit() {
		// Guard against a second Enter press (CodeInput.onsubmit) firing while a request is in flight.
		if (isLoading || !codeComplete) return;
		isLoading = true;
		errorMessage = null;
		try {
			deviceInfo = await oidcService.getDeviceCodeInfo(userCode);

			if (deviceInfo.authorizationRequired) {
				modalState = 'authorize-scopes';
				return;
			}

			await oidcService.verifyDeviceCode(userCode);
			modalState = 'success';
		} catch (e) {
			errorMessage = getAxiosErrorMessage(e);
			modalState = 'error';
		} finally {
			isLoading = false;
		}
	}

	async function authorizeScopes() {
		if (isLoading) return;
		isLoading = true;
		try {
			await oidcService.verifyDeviceCode(userCode);
			modalState = 'success';
		} catch (e) {
			errorMessage = getAxiosErrorMessage(e);
			modalState = 'error';
		} finally {
			isLoading = false;
		}
	}

	function reset() {
		modalState = 'enter';
		isLoading = false;
		userCode = '';
		errorMessage = null;
		deviceInfo = undefined;
	}

	function onOpenChange(open: boolean) {
		if (!open) {
			show = false;
			reset();
		}
	}
</script>

<Dialog.Root open={show} {onOpenChange}>
	<Dialog.Content class="max-w-md">
		<Dialog.Header>
			<Dialog.Title>{m.enter_code()}</Dialog.Title>
			<Dialog.Description>{m.enter_code_displayed_in_previous_step()}</Dialog.Description>
		</Dialog.Header>

		{#if modalState === 'enter'}
			<div class="flex flex-col items-center gap-4 py-2">
				<CodeInput bind:value={userCode} autofocus onsubmit={submit} />
				<Button class="w-full" disabled={!codeComplete} onclick={submit} {isLoading}>
					{m.authorize()}
				</Button>
			</div>
		{:else if modalState === 'authorize-scopes'}
			<div class="flex flex-col items-center gap-4 py-2">
				<Card.Root class="w-full">
					<Card.Header class="pb-3">
						<p class="text-muted-foreground text-sm">
							<FormattedMessage
								m={m.client_wants_to_access_the_following_information({
									client: deviceInfo!.client.name
								})}
							/>
						</p>
					</Card.Header>
					<Card.Content>
						<ScopeList scope={deviceInfo!.scope} />
					</Card.Content>
				</Card.Root>
				<div class="flex w-full gap-2">
					<Button class="flex-1" variant="secondary" onclick={reset}>{m.cancel()}</Button>
					<Button class="flex-1" onclick={authorizeScopes} {isLoading}>
						{m.authorize()}
					</Button>
				</div>
			</div>
		{:else if modalState === 'success'}
			<p class="text-muted-foreground py-4 text-center">
				{m.the_device_has_been_authorized()}
			</p>
		{:else if modalState === 'error'}
			<div class="flex flex-col items-center gap-4 py-2">
				<p class="text-destructive text-center text-sm">{errorMessage}</p>
				<Button class="w-full" variant="outline" onclick={reset}>{m.try_again()}</Button>
			</div>
		{/if}
	</Dialog.Content>
</Dialog.Root>
