<script lang="ts">
	import CodeInput from '$lib/components/code-input.svelte';
	import CopyToClipboard from '$lib/components/copy-to-clipboard.svelte';
	import QRCode from '$lib/components/qrcode/qrcode.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Spinner } from '$lib/components/ui/spinner';
	import { m } from '$lib/paraglide/messages';
	import OidcService from '$lib/services/oidc-service';
	import { onMount } from 'svelte';

	let { onauthorized }: { onauthorized: () => void } = $props();

	const oidcService = new OidcService();

	const POLL_INTERVAL_INIT_S = 5;
	const POLL_INTERVAL_MAX_S = 60;
	const POLL_INTERVAL_INCREMENT_S = 5;

	type FlowState = 'loading' | 'showing' | 'authorized' | 'expired' | 'error';

	let flowState: FlowState = $state('loading');
	let userCode: string = $state('');
	let verificationUriComplete: string = $state('');
	let deviceCode: string = $state('');
	let pollingInterval: number = $state(POLL_INTERVAL_INIT_S);
	let remainingSeconds: number = $state(0);
	let expiresAt: number = $state(0);
	let errorMessage: string = $state('');
	let abortController: AbortController | null = null;
	let countdownInterval: ReturnType<typeof setInterval> | null = null;

	let formattedTime = $derived(
		`${String(Math.floor(remainingSeconds / 60)).padStart(2, '0')}:${String(remainingSeconds % 60).padStart(2, '0')}`
	);

	onMount(() => {
		startFlow();
		return () => cleanup();
	});

	function cleanup() {
		abortController?.abort();
		abortController = null;
		if (countdownInterval) {
			clearInterval(countdownInterval);
			countdownInterval = null;
		}
	}

	function syncRemainingSeconds() {
		// Math.ceil so the initial display shows the full duration (e.g. 5:00 not 4:59).
		const remaining = Math.max(0, Math.ceil((expiresAt - Date.now()) / 1000));
		remainingSeconds = remaining;
		if (remaining <= 0) {
			// Stop the interval unconditionally, regardless of current flowState. Otherwise it keeps
			// firing forever after flowState becomes 'authorized' (countdown leak).
			if (countdownInterval) {
				clearInterval(countdownInterval);
				countdownInterval = null;
			}
			if (flowState === 'showing') {
				flowState = 'expired';
				abortController?.abort();
				abortController = null;
			}
		}
	}

	async function startFlow() {
		cleanup();
		flowState = 'loading';
		errorMessage = '';
		userCode = '';
		verificationUriComplete = '';
		deviceCode = '';
		pollingInterval = POLL_INTERVAL_INIT_S;
		remainingSeconds = 0;

		try {
			const data = await oidcService.createSelfLoginDeviceCode();
			deviceCode = data.device_code;
			userCode = data.user_code;
			verificationUriComplete = data.verification_uri_complete;
			pollingInterval = data.interval || POLL_INTERVAL_INIT_S;
			expiresAt = Date.now() + data.expires_in * 1000;

			syncRemainingSeconds();
			countdownInterval = setInterval(syncRemainingSeconds, 1000);

			flowState = 'showing';

			abortController = new AbortController();
			pollAndExchange(abortController.signal);
		} catch (e: any) {
			cleanup();
			errorMessage = e?.message || m.qr_login_failed_to_create_code();
			flowState = 'error';
		}
	}

	function waitWithAbort(ms: number, signal: AbortSignal): Promise<void> {
		return new Promise<void>((resolve, reject) => {
			let timer: ReturnType<typeof setTimeout> | null = null;
			const onAbort = () => {
				if (timer) clearTimeout(timer);
				reject(new Error('Polling aborted'));
			};
			timer = setTimeout(() => {
				signal.removeEventListener('abort', onAbort);
				resolve();
			}, ms);
			signal.addEventListener('abort', onAbort, { once: true });
		});
	}

	async function pollAndExchange(signal: AbortSignal) {
		while (true) {
			if (signal.aborted) return;
			await waitWithAbort(pollingInterval * 1000, signal).catch(() => {});
			if (signal.aborted) return;

			try {
				await oidcService.exchangeDeviceTokenForSession(deviceCode);
				if (signal.aborted) return;
				cleanup();
				flowState = 'authorized';
				onauthorized();
				return;
			} catch (e: any) {
				if (signal.aborted) return;
				const errorCode = e.response?.data?.error;
				if (errorCode === 'authorization_pending') continue;
				if (errorCode === 'slow_down') {
					// RFC 8628 §3.5: interval MUST be increased by at least 5s; never decrease.
					const serverInterval =
						typeof e.response?.data?.interval === 'number' ? e.response.data.interval : 0;
					pollingInterval = Math.min(
						POLL_INTERVAL_MAX_S,
						Math.max(pollingInterval + POLL_INTERVAL_INCREMENT_S, serverInterval)
					);
					continue;
				}
				if (errorCode === 'expired_token') {
					cleanup();
					flowState = 'expired';
					return;
				}
				if (errorCode === 'access_denied') {
					cleanup();
					errorMessage = m.qr_login_access_denied();
					flowState = 'error';
					return;
				}
				if (errorCode === 'invalid_grant') {
					cleanup();
					errorMessage = m.qr_login_invalid_grant();
					flowState = 'error';
					return;
				}
				cleanup();
				errorMessage = e?.message || m.qr_login_unexpected_error();
				flowState = 'error';
				return;
			}
		}
	}
</script>

<div class="flex flex-col items-center gap-4">
	{#if flowState === 'loading'}
		<div class="flex items-center justify-center py-10">
			<Spinner class="size-8" />
		</div>
	{:else if flowState === 'showing'}
		<div class="rounded-lg bg-white p-3">
			<QRCode value={verificationUriComplete} size={200} />
		</div>
		<p class="text-muted-foreground text-sm">
			{m.scan_qr_code_or_enter_code_manually()}
		</p>
		<CopyToClipboard value={userCode}>
			<CodeInput value={userCode} disabled />
		</CopyToClipboard>
		<p class="text-muted-foreground text-sm">
			{m.expires_in_time({ time: formattedTime })}
		</p>
	{:else if flowState === 'expired'}
		<p class="text-muted-foreground">{m.code_has_expired()}</p>
		<Button onclick={startFlow}>{m.refresh()}</Button>
	{:else if flowState === 'authorized'}
		<p class="text-muted-foreground">{m.authorized_redirecting()}</p>
	{:else if flowState === 'error'}
		<p class="text-destructive">{errorMessage}</p>
		<Button onclick={startFlow}>{m.try_again()}</Button>
	{/if}
</div>
