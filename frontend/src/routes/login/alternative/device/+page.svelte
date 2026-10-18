<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import AuthHeading from '#lib/components/auth/auth-heading.svelte';
	import AuthTile from '#lib/components/auth/auth-tile.svelte';
	import {
		authPrimaryClass,
		authQuietClass,
		successHold,
		takingYouMessage
	} from '#lib/components/auth/auth-util.ts';
	import SignInWrapper from '#lib/components/login-wrapper.svelte';
	import QrPanel from '#lib/components/qrcode/qr-panel.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { Spinner } from '#lib/components/ui/spinner/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import DeviceLoginService from '#lib/services/device-login-service.ts';
	import userStore from '#lib/stores/user-store.ts';
	import type { DeviceLoginRequest } from '#lib/types/device-login.type.ts';
	import { getAxiosErrorCode, getAxiosErrorMessage } from '#lib/utils/error-util.ts';
	import { isAxiosError } from 'axios';
	import { onMount } from 'svelte';

	let { data } = $props();

	const deviceLoginService = new DeviceLoginService();

	let request: DeviceLoginRequest | undefined = $state();
	let ended: { kind: 'expired' | 'denied' | 'failed'; message?: string } | undefined = $state();
	let success = $state(false);
	let pollTimer: ReturnType<typeof setTimeout> | undefined;
	let requestController: AbortController | undefined;

	onMount(() => {
		void startRequest();

		return () => {
			stopRequest();
		};
	});

	async function startRequest() {
		stopRequest();
		// Track this attempt so stale responses cannot update the current page
		const controller = new AbortController();
		requestController = controller;

		try {
			const next = await deviceLoginService.createRequest(controller.signal);
			// Ignore a response from an attempt replaced while the request was in flight
			if (requestController !== controller) return;
			// The old panel stays until the new code arrives, so the card does not collapse in between
			request = next;
			ended = undefined;
			schedulePoll(controller);
		} catch (error) {
			if (controller.signal.aborted) return;
			ended = { kind: 'failed', message: getAxiosErrorMessage(error) };
		}
	}

	function schedulePoll(controller: AbortController) {
		// Ignore schedules from stale attempts or before request metadata exists
		if (!request || requestController !== controller) return;
		// Delay the next exchange so the client respects the server-provided polling interval
		pollTimer = setTimeout(() => void exchangeRequest(controller), request.interval * 1000);
	}

	async function exchangeRequest(controller: AbortController) {
		// Ignore work from a stopped or superseded attempt
		if (!request || requestController !== controller) return;

		try {
			const user = await deviceLoginService.exchangeRequest(request.id, controller.signal);
			// Ignore a response from an attempt replaced while the exchange was in flight
			if (requestController !== controller) return;
			if (!user) {
				schedulePoll(controller);
				return;
			}

			clearTimers();
			await userStore.setUser(user);
			// Navigate only after confirming that this attempt is still active
			if (requestController !== controller) return;
			success = true;
			await successHold();
			await goto(data.redirect);
		} catch (error) {
			if (controller.signal.aborted) return;
			// Retry transport and proxy failures because the device login request remains pending
			if (isRetryableDeviceLoginExchangeError(error)) {
				schedulePoll(controller);
				return;
			}
			// Stop retrying when the server reports a non-transient failure
			clearTimers();

			const code = getAxiosErrorCode(error);
			if (code === 'device_login_denied') ended = { kind: 'denied' };
			else if (code === 'device_login_expired') ended = { kind: 'expired' };
			else ended = { kind: 'failed', message: getAxiosErrorMessage(error) };
		}
	}

	function isRetryableDeviceLoginExchangeError(error: unknown) {
		// Ignore non-Axios errors and intentional cancellations
		if (!isAxiosError(error) || error.code === 'ERR_CANCELED') return false;

		const retryableExchangeStatuses = new Set([408, 425, 429, 502, 503, 504, 522, 524]);
		// Retry failures without an HTTP response or known transient gateway responses
		return (
			error.response?.status === undefined || retryableExchangeStatuses.has(error.response.status)
		);
	}

	function stopRequest() {
		// Abort the active exchange and clear its timer before replacing or leaving it
		requestController?.abort();
		requestController = undefined;
		clearTimers();
	}

	function clearTimers() {
		// Ensure an old scheduled poll cannot start after the request lifecycle ends
		if (pollTimer) clearTimeout(pollTimer);
		pollTimer = undefined;
	}
</script>

<svelte:head>
	<title>{m.sign_in_with_another_device()}</title>
</svelte:head>

<SignInWrapper back={'/login/alternative' + page.url.search}>
	<AuthTile {success} />
	<AuthHeading
		title={success ? m.auth_signed_in() : m.sign_in_with_another_device()}
		tone={ended ? (ended.kind === 'failed' ? 'danger' : 'info') : undefined}
	>
		{#if success}
			{takingYouMessage(data.redirect)}
		{:else if ended?.kind === 'expired'}
			{m.auth_another_device_expired()}
		{:else if ended?.kind === 'denied'}
			{m.auth_another_device_denied()}
		{:else if ended}
			{ended.message}
		{:else}
			{m.auth_use_another_device_description()}
		{/if}
	</AuthHeading>
	<div
		class="flex flex-col items-center transition-opacity duration-(--dur-fast)"
		class:opacity-0={success}
		inert={success}
	>
		{#if request}
			<div class="mb-6 w-full">
				<QrPanel {request} ended={!!ended} />
			</div>
		{/if}
		{#if ended}
			<Button class={authPrimaryClass} usePromiseLoading onclick={startRequest}>
				{ended.kind === 'denied' ? m.try_again() : m.auth_get_a_new_code()}
			</Button>
		{:else}
			<!-- While the request is alive there is nothing to press, so the primary's place holds the wait as a status line -->
			<p
				class="text-body flex min-h-(--h-xl) items-center justify-center gap-2 text-center lg:min-h-13"
				aria-live="polite"
			>
				<Spinner class="size-3 shrink-0" />
				{request ? m.auth_waiting_for_approval() : m.auth_creating_sign_in_request()}
			</p>
			<Button
				variant="link"
				class={authQuietClass + ' mt-3'}
				disabled={!request}
				onclick={startRequest}
			>
				{m.auth_get_a_new_code()}
			</Button>
		{/if}
	</div>
</SignInWrapper>
