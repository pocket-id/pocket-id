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

	type ModalState = 'enter' | 'loading' | 'authorize-scopes' | 'success' | 'error';

	let state: ModalState = $state('enter');
	let userCode = $state('');
	let errorMessage: string | null = $state(null);
	let deviceInfo: OidcDeviceCodeInfo | undefined = $state();

	let codeComplete = $derived(userCode.replace(/[^a-zA-Z0-9]/g, '').length >= 8);

	async function submit() {
		if (!codeComplete) return;
		state = 'loading';
		errorMessage = null;
		try {
			deviceInfo = await oidcService.getDeviceCodeInfo(userCode);

			if (deviceInfo.authorizationRequired) {
				state = 'authorize-scopes';
				return;
			}

			await oidcService.verifyDeviceCode(userCode);
			state = 'success';
		} catch (e) {
			errorMessage = getAxiosErrorMessage(e);
			state = 'error';
		}
	}

	async function authorizeScopes() {
		state = 'loading';
		try {
			await oidcService.verifyDeviceCode(userCode);
			state = 'success';
		} catch (e) {
			errorMessage = getAxiosErrorMessage(e);
			state = 'error';
		}
	}

	function reset() {
		state = 'enter';
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

		{#if state === 'enter' || (state === 'loading' && !deviceInfo)}
			<div class="flex flex-col items-center gap-4 py-2">
				<CodeInput bind:value={userCode} autofocus onsubmit={submit} />
				<Button
					class="w-full"
					disabled={!codeComplete}
					onclick={submit}
					isLoading={state === 'loading'}
				>
					{m.authorize()}
				</Button>
			</div>
		{:else if state === 'authorize-scopes'}
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
					<Button class="flex-1" onclick={authorizeScopes} isLoading={state === 'loading'}>
						{m.authorize()}
					</Button>
				</div>
			</div>
		{:else if state === 'success'}
			<p class="text-muted-foreground py-4 text-center">
				{m.the_device_has_been_authorized()}
			</p>
		{:else if state === 'error'}
			<div class="flex flex-col items-center gap-4 py-2">
				<p class="text-destructive text-center text-sm">{errorMessage}</p>
				<Button class="w-full" variant="outline" onclick={reset}>{m.try_again()}</Button>
			</div>
		{/if}
	</Dialog.Content>
</Dialog.Root>
