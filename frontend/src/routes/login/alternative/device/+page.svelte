<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import AuthHeading from '#lib/components/auth/auth-heading.svelte';
	import AuthTile from '#lib/components/auth/auth-tile.svelte';
	import {
		authPrimaryClass,
		successHold,
		takingYouMessage
	} from '#lib/components/auth/auth-util.ts';
	import SignInWrapper from '#lib/components/login-wrapper.svelte';
	import Qrcode from '#lib/components/qrcode/qrcode.svelte';
	import { Button } from '@pocket-id/ui/button';
	import { Spinner } from '@pocket-id/ui/spinner';
	import { m } from '#lib/paraglide/messages.js';
	import DeviceLoginService from '#lib/services/device-login-service.ts';
	import userStore from '#lib/stores/user-store.ts';
	import type { DeviceLoginRequest } from '#lib/types/device-login.type.ts';
	import { getAxiosErrorCode, getAxiosErrorMessage } from '#lib/utils/error-util.ts';
	import { cn } from '@pocket-id/ui/utils';
	import { isAxiosError } from 'axios';
	import { onMount } from 'svelte';
	import CheckIcon from '~icons/solar/check-linear';
	import CopyIcon from '~icons/solar/copy-bold';

	let { data } = $props();

	const deviceLoginService = new DeviceLoginService();

	let request: DeviceLoginRequest | undefined = $state();
	let ended: { kind: 'expired' | 'denied' | 'failed'; message?: string } | undefined = $state();
	let success = $state(false);
	let pollTimer: ReturnType<typeof setTimeout> | undefined;
	let copied = $state(false);
	let copiedTimer: ReturnType<typeof setTimeout> | undefined;

	let now = $state(Date.now());
	$effect(() => {
		const timer = setInterval(() => (now = Date.now()), 1000);
		return () => clearInterval(timer);
	});

	const expiresAt = $derived(new Date(request?.expiresAt ?? 0).getTime());
	// Reads the clock only when the request changes, so the ring measures against the lifetime the code started with
	const lifetime = $derived(Math.max(1, expiresAt - Date.now()));
	const secondsLeft = $derived(Math.max(0, Math.ceil((expiresAt - now) / 1000)));
	// The server can end the request before the local countdown does, and a denied code is just as dead
	const expired = $derived(!!ended || (!!request && secondsLeft === 0));
	const remaining = $derived(expired ? 0 : (secondsLeft * 1000) / lifetime);
	const verificationUrl = $derived(request?.verificationUri.replace(/^https?:\/\//, '') ?? '');
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

	async function copyCode() {
		await navigator.clipboard.writeText(request!.userCode);
		copied = true;
		clearTimeout(copiedTimer);
		copiedTimer = setTimeout(() => (copied = false), 1500);
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
		{:else if request}
			<!-- Phones show only the code, so the subtitle carries the instruction instead of a second line above it -->
			<span class="max-sm:hidden">{m.auth_use_another_device_description()}</span>
			<span class="sm:hidden">{m.device_login_go_to({ url: verificationUrl })}</span>
		{:else}
			{m.auth_use_another_device_description()}
		{/if}
	</AuthHeading>
	<div
		class="flex flex-col items-center transition-opacity duration-(--dur-fast) max-sm:items-stretch"
		class:opacity-0={success}
		inert={success}
	>
		{#if request}
			<!-- A phone cannot scan its own screen, so phones get only the code -->
			<!-- The ring drains over the code's lifetime, as the bar under the code does on phones, so neither needs countdown text -->
			<!-- It shrinks on short windows so the card fits without scrolling for as long as it can -->
			<div
				class="relative mb-6 grid size-(--ring) place-items-center [--ring:clamp(176px,100dvh-630px,232px)] max-sm:hidden"
			>
				<svg class="absolute inset-0" viewBox="0 0 232 232" aria-hidden="true">
					<rect
						x="2"
						y="2"
						width="228"
						height="228"
						rx="26"
						fill="none"
						stroke="var(--border)"
						stroke-width="3"
					/>
					<rect
						x="2"
						y="2"
						width="228"
						height="228"
						rx="26"
						fill="none"
						stroke="var(--fg)"
						stroke-width="3"
						stroke-linecap="round"
						pathLength="100"
						stroke-dasharray="100"
						stroke-dashoffset={100 * (1 - remaining)}
						class="transition-[stroke-dashoffset] duration-1000 ease-linear"
					/>
				</svg>
				<Qrcode
					value={request.verificationUriComplete}
					size={176}
					role="img"
					aria-label={m.device_login_qr_code()}
					aria-hidden={expired}
					class={cn(
						'h-auto w-[calc(var(--ring)-56px)] rounded-[18px] p-4 transition-opacity duration-(--dur-base)',
						expired && 'opacity-20'
					)}
				/>
			</div>
			<p class="text-body text-fg-muted mb-3 text-center wrap-anywhere max-sm:hidden">
				{m.device_login_or_go_to({ url: verificationUrl })}
			</p>
			<button
				type="button"
				class={cn(
					'bg-surface-sunken hover:bg-fill-hover inset-ring-border disabled:hover:bg-surface-sunken mb-6 flex items-center justify-center gap-3 rounded-full py-2.5 pr-4 pl-6 font-mono text-2xl font-medium tracking-[0.12em] inset-ring transition-colors disabled:cursor-default max-sm:mb-3 max-sm:rounded-lg max-sm:py-5 max-sm:text-3xl',
					expired && 'text-fg-muted'
				)}
				disabled={expired}
				aria-label={m.copy_code()}
				onclick={copyCode}
				data-testid="device-login-code"
			>
				{request.userCode.slice(0, 4)}–{request.userCode.slice(4)}
				{#if !expired}
					<span class="text-fg-muted grid size-4 *:col-start-1 *:row-start-1" aria-hidden="true">
						<CopyIcon class={cn('size-4 transition-opacity', copied && 'opacity-0')} />
						<CheckIcon class={cn('size-4 transition-opacity', !copied && 'opacity-0')} />
					</span>
				{/if}
			</button>
			<div class="bg-border mb-6 h-1 overflow-hidden rounded-full sm:hidden" aria-hidden="true">
				<div
					class="bg-fg h-full origin-left rounded-full transition-transform duration-1000 ease-linear"
					style:scale="{remaining} 1"
				></div>
			</div>
			<span class="sr-only" aria-live="polite">{copied ? m.copied() : ''}</span>
		{/if}
		{#if expired}
			<Button class={authPrimaryClass} usePromiseLoading onclick={startRequest}>
				{ended?.kind === 'denied' ? m.try_again() : m.auth_get_a_new_code()}
			</Button>
		{:else}
			<p class="text-control text-fg-muted flex items-center gap-2" aria-live="polite">
				<Spinner class="size-3 shrink-0" />
				{request ? m.auth_waiting_for_approval() : m.auth_creating_sign_in_request()}
			</p>
		{/if}
	</div>
</SignInWrapper>
