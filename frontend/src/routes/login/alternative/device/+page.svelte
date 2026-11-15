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
	import { m } from '#lib/paraglide/messages.js';
	import DeviceLoginService from '#lib/services/device-login-service.ts';
	import { keepSignedIn } from '#lib/stores/keep-signed-in-store.svelte.ts';
	import userStore from '#lib/stores/user-store.ts';
	import type { DeviceLoginRequest } from '#lib/types/device-login.type.ts';
	import { getAxiosErrorCode, getAxiosErrorMessage } from '#lib/utils/error-util.ts';
	import { cssMs } from '#lib/utils/style.ts';
	import { cn } from '@pocket-id/ui/utils';
	import { isAxiosError } from 'axios';
	import { onMount } from 'svelte';
	import CheckIcon from '~icons/solar/check-linear';
	import CopyIcon from '~icons/solar/copy-bold';
	import { fade } from 'svelte/transition';

	let { data } = $props();

	const deviceLoginService = new DeviceLoginService();

	let request: DeviceLoginRequest | undefined = $state();
	let ended: { kind: 'expired' | 'denied' | 'failed'; message?: string } | undefined = $state();
	let success = $state(false);
	let pollTimer: ReturnType<typeof setTimeout> | undefined;
	let copied: 'code' | 'url' | undefined = $state();
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
	const active = $derived(!!request && !expired && !success);
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
			await keepSignedIn.ask();
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

	async function copy(what: 'code' | 'url') {
		await navigator.clipboard.writeText(
			what === 'code' ? request!.userCode : request!.verificationUri
		);
		copied = what;
		clearTimeout(copiedTimer);
		copiedTimer = setTimeout(() => (copied = undefined), 1500);
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

{#snippet copyState(done: boolean)}
	<span class="text-fg-muted grid size-4 shrink-0 *:col-start-1 *:row-start-1" aria-hidden="true">
		<CopyIcon class={cn('size-4 transition-opacity', done && 'opacity-0')} />
		<CheckIcon class={cn('size-4 transition-opacity', !done && 'opacity-0')} />
	</span>
{/snippet}

{#snippet scan()}
	<!-- A phone cannot scan its own screen, and on narrow cards this column follows the typing instructions -->
	<div
		class="bg-surface-sunken mb-6 flex flex-col items-center gap-4 rounded-xl p-6 text-center max-sm:hidden lg:mb-0 lg:h-full lg:justify-center"
		out:fade={{ duration: cssMs('--dur-fast') }}
	>
		<!-- The ring drains over the code's lifetime -->
		<div class="relative grid size-58 place-items-center">
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
				value={request!.verificationUriComplete}
				size={144}
				role="img"
				aria-label={m.device_login_qr_code()}
				class="rounded-[18px] p-4"
			/>
		</div>
		<div class="flex max-w-64 flex-col gap-1">
			<p class="text-heading">{m.device_login_scan_title()}</p>
			<p class="text-body text-fg-muted">{m.device_login_scan_description()}</p>
		</div>
	</div>
{/snippet}

<SignInWrapper
	back={'/login/alternative' + page.url.search}
	wide={active}
	side={active ? scan : undefined}
>
	<AuthTile {success} error={ended?.kind === 'failed'} />
	<AuthHeading title={success ? m.auth_signed_in() : m.sign_in_with_another_device()}>
		{#if success}
			{takingYouMessage(data.redirect)}
		{:else if ended?.kind === 'denied'}
			{m.auth_another_device_denied()}
		{:else if ended?.kind === 'failed'}
			{ended.message}
		{:else if expired}
			{m.auth_another_device_expired()}
		{:else}
			{m.auth_use_another_device_description()}
		{/if}
	</AuthHeading>
	{#if active}
		<div class="mb-6 flex flex-col gap-5" out:fade={{ duration: cssMs('--dur-fast') }}>
			<div class="flex flex-col items-start gap-1.5">
				<p class="text-caption text-fg-muted">{m.device_login_open_on_that_device()}</p>
				<button
					type="button"
					class="hover:text-fg-muted text-title flex max-w-full items-center gap-2 text-left font-medium transition-colors"
					aria-label={m.copy_link()}
					onclick={() => copy('url')}
				>
					<span class="wrap-anywhere">{verificationUrl}</span>
					{@render copyState(copied === 'url')}
				</button>
			</div>
			<div class="flex flex-col items-start gap-1.5">
				<p class="text-caption text-fg-muted">{m.device_login_and_enter()}</p>
				<button
					type="button"
					class="hover:text-fg-muted flex items-center gap-3 font-mono text-3xl font-medium tracking-[0.12em] transition-colors"
					aria-label={m.copy_code()}
					onclick={() => copy('code')}
					data-testid="device-login-code"
				>
					{request!.userCode.slice(0, 4)}–{request!.userCode.slice(4)}
					{@render copyState(copied === 'code')}
				</button>
			</div>
			<p class="text-caption text-fg-muted">
				{m.device_login_expires_in({
					time: `${Math.floor(secondsLeft / 60)}:${String(secondsLeft % 60).padStart(2, '0')}`
				})}
			</p>
		</div>
		<span class="sr-only" aria-live="polite">{copied ? m.copied() : ''}</span>
	{:else if expired && !success}
		<Button class={authPrimaryClass} usePromiseLoading onclick={startRequest}>
			{ended?.kind === 'denied' ? m.try_again() : m.auth_get_a_new_code()}
		</Button>
	{/if}
</SignInWrapper>
