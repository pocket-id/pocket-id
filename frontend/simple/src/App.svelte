<script lang="ts">
	import { onDestroy, onMount, tick } from 'svelte';

	const SELF_LOGIN_CLIENT_ID = 'pocket-id-self-login';
	const SELF_LOGIN_SCOPE = 'openid profile email';
	const API_BASE = '/api/oidc/device';

	// Standalone bundle, no shared imports: keep in sync with qr-login-flow.svelte.
	const POLL_INTERVAL_INIT_S = 5;
	const POLL_INTERVAL_MAX_S = 60;
	const POLL_INTERVAL_INCREMENT_S = 5;
	const REQUEST_TIMEOUT_MS = 15000;
	const REQUEST_RETRY_DELAY_MS = 2000;
	const AUTHORIZED_REDIRECT_DELAY_MS = 3000;
	// Fallback only; the server always sends expires_in. Matches the backend DeviceCodeDuration (15 min).
	const DEFAULT_EXPIRES_IN_S = 900;
	const NETWORK_ERROR_MAX_RETRIES = 5;
	const TV_BREAKPOINT_PX = 1200;
	const QR_SIZE_PX = 200;
	const QR_SIZE_TV_PX = 320;

	type View = 'loading' | 'showing' | 'authorized' | 'expired' | 'error';

	let view: View = $state('loading');
	let appName = $state('Sign In');
	let userCodeChars: string[] = $state([]);
	let countdown = $state('');
	let errorMessage = $state('');

	let deviceCode = '';
	let pollingInterval = POLL_INTERVAL_INIT_S;
	let expiresAt = 0;
	let pollTimer: ReturnType<typeof setTimeout> | null = null;
	let countdownTimer: ReturnType<typeof setInterval> | null = null;
	let redirectTimer: ReturnType<typeof setTimeout> | null = null;
	let redirectUrl = '';
	let pendingQrUrl = '';
	let stateEl: HTMLElement | null = null;
	let activeXhr: XMLHttpRequest | null = null;
	let networkErrorCount = 0;
	let flowId = 0;

	$effect(() => {
		if (view === 'showing' && pendingQrUrl) {
			const url = pendingQrUrl;
			pendingQrUrl = '';
			tick().then(() => renderQR(url));
		}
	});

	$effect(() => {
		if (view !== 'loading' && stateEl) {
			stateEl.focus();
		}
	});

	// Mirrors redirection-util.ts (standalone bundle, no imports).
	function isSafeRedirect(url: string): boolean {
		return !!url && url.startsWith('/') && !url.startsWith('//') && !url.startsWith('/\\');
	}

	function getRedirectParam(): string {
		const url = new URLSearchParams(window.location.search).get('redirect') ?? '';
		return isSafeRedirect(url) ? url : '';
	}

	function formatTime(seconds: number): string {
		const m = Math.floor(seconds / 60);
		const s = seconds % 60;
		return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`;
	}

	function cleanup() {
		if (pollTimer) {
			clearTimeout(pollTimer);
			pollTimer = null;
		}
		if (countdownTimer) {
			clearInterval(countdownTimer);
			countdownTimer = null;
		}
		if (redirectTimer) {
			clearTimeout(redirectTimer);
			redirectTimer = null;
		}
		if (activeXhr) {
			try {
				activeXhr.abort();
			} catch {
				/* ignore */
			}
			activeXhr = null;
		}
	}

	function updateCountdown() {
		// Math.ceil so the initial display shows full duration (e.g. 5:00 not 4:59).
		const remaining = Math.max(0, Math.ceil((expiresAt - Date.now()) / 1000));
		countdown = `Code expires in ${formatTime(remaining)}`;
		if (remaining <= 0) {
			cleanup();
			view = 'expired';
		}
	}

	function renderQR(url: string) {
		const canvas = document.getElementById('qr-canvas') as HTMLCanvasElement | null;
		if (!canvas || !url) return;
		const QRCode = (window as any).QRCode;
		if (!QRCode) {
			console.error('QRCode library not loaded');
			return;
		}
		const size = window.innerWidth >= TV_BREAKPOINT_PX ? QR_SIZE_TV_PX : QR_SIZE_PX;
		QRCode.toCanvas(canvas, url, { width: size, margin: 0 }, (err: unknown) => {
			if (err) console.error('QR render failed:', err);
		});
	}

	// XHR rather than fetch: older Smart-TV / WebOS browsers lack reliable fetch()/AbortController.
	// One-shot calls pass retries=1; the polling loop passes 0 and uses its own backoff instead.
	function request(
		method: string,
		url: string,
		data: string | null,
		callback: (status: number, data: any) => void,
		retries = 1
	) {
		const xhr = new XMLHttpRequest();
		activeXhr = xhr;
		xhr.open(method, url, true);
		xhr.timeout = REQUEST_TIMEOUT_MS;
		if (method === 'POST') {
			xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');
		}

		let settled = false;
		const settle = (status: number, body: any) => {
			if (settled) return;
			settled = true;
			if (activeXhr === xhr) activeXhr = null;
			callback(status, body);
		};

		function onError() {
			if (settled) return;
			if (retries > 0) {
				settled = true;
				if (activeXhr === xhr) activeXhr = null;
				setTimeout(() => request(method, url, data, callback, retries - 1), REQUEST_RETRY_DELAY_MS);
			} else {
				settle(0, null);
			}
		}

		xhr.ontimeout = onError;
		xhr.onerror = onError;
		xhr.onabort = () => {
			settled = true;
			if (activeXhr === xhr) activeXhr = null;
		};
		xhr.onreadystatechange = () => {
			if (xhr.readyState !== 4) return;
			// Some Smart-TV browsers (WebOS/Tizen) deliver readyState=4 with status=0 on network
			// failure WITHOUT firing onerror/ontimeout. Treat that path as a network error so the
			// caller can apply backoff instead of hanging forever.
			if (xhr.status === 0) {
				onError();
				return;
			}
			let response = null;
			try {
				response = JSON.parse(xhr.responseText);
			} catch {
				/* ignore */
			}
			settle(xhr.status, response);
		};

		xhr.send(data);
	}

	function poll(myFlowId: number) {
		if (myFlowId !== flowId) return;
		const params = `device_code=${encodeURIComponent(deviceCode)}&client_id=${encodeURIComponent(SELF_LOGIN_CLIENT_ID)}`;

		request(
			'POST',
			`${API_BASE}/exchange-session`,
			params,
			(status, data) => {
				if (myFlowId !== flowId) return;

				if (status >= 200 && status < 300) {
					cleanup();
					view = 'authorized';
					redirectTimer = setTimeout(() => {
						if (myFlowId !== flowId) return;
						window.location.href = redirectUrl || '/';
					}, AUTHORIZED_REDIRECT_DELAY_MS);
					return;
				}

				// Any response with a parsed body counts as "server reachable" → reset network counter.
				if (data) networkErrorCount = 0;

				if (data?.error) {
					if (data.error === 'authorization_pending') {
						pollTimer = setTimeout(() => poll(myFlowId), pollingInterval * 1000);
						return;
					}
					if (data.error === 'slow_down') {
						// RFC 8628 §3.5: interval MUST be increased by at least 5s; never decrease.
						const serverInterval = typeof data.interval === 'number' ? data.interval : 0;
						pollingInterval = Math.min(
							POLL_INTERVAL_MAX_S,
							Math.max(pollingInterval + POLL_INTERVAL_INCREMENT_S, serverInterval)
						);
						pollTimer = setTimeout(() => poll(myFlowId), pollingInterval * 1000);
						return;
					}
					if (data.error === 'expired_token') {
						cleanup();
						view = 'expired';
						return;
					}
					if (data.error === 'access_denied') {
						cleanup();
						errorMessage = 'Login was denied on the other device.';
						view = 'error';
						return;
					}
					if (data.error === 'invalid_grant') {
						cleanup();
						errorMessage = 'Invalid or unknown device code. Please try again.';
						view = 'error';
						return;
					}
				}

				if (status === 0) {
					networkErrorCount++;
					if (networkErrorCount >= NETWORK_ERROR_MAX_RETRIES) {
						cleanup();
						errorMessage = 'Cannot reach the server. Please check your connection and try again.';
						view = 'error';
						return;
					}
					// Exponential backoff for transient network failures, separate from the RFC 8628
					// slow_down interval; not persisted into pollingInterval.
					const backoff = Math.min(
						POLL_INTERVAL_MAX_S,
						pollingInterval * Math.pow(2, networkErrorCount - 1)
					);
					pollTimer = setTimeout(() => poll(myFlowId), backoff * 1000);
					return;
				}

				cleanup();
				if (data?.error) console.error('Device flow error:', data.error);
				errorMessage = 'An unexpected error occurred. Please try again.';
				view = 'error';
			},
			0
		);
	}

	function startFlow() {
		cleanup();
		flowId++;
		const myFlowId = flowId;

		// Reset all flow-scoped state so a retry never inherits stale values.
		view = 'loading';
		userCodeChars = [];
		countdown = '';
		errorMessage = '';
		deviceCode = '';
		pollingInterval = POLL_INTERVAL_INIT_S;
		expiresAt = 0;
		pendingQrUrl = '';
		networkErrorCount = 0;

		const params = `client_id=${encodeURIComponent(SELF_LOGIN_CLIENT_ID)}&scope=${encodeURIComponent(SELF_LOGIN_SCOPE)}`;

		request('POST', `${API_BASE}/authorize`, params, (status, data) => {
			if (myFlowId !== flowId) return;

			if (status < 200 || status >= 300 || !data) {
				if (data?.error) console.error('Device authorize error:', data.error);
				errorMessage = 'Failed to start the login. Please try again.';
				view = 'error';
				return;
			}

			if (!data.device_code || !data.user_code || !data.verification_uri_complete) {
				errorMessage = 'Invalid response from server.';
				view = 'error';
				return;
			}

			deviceCode = data.device_code;
			pollingInterval = data.interval ?? POLL_INTERVAL_INIT_S;
			expiresAt = Date.now() + (data.expires_in ?? DEFAULT_EXPIRES_IN_S) * 1000;
			userCodeChars = (data.user_code as string).split('');
			pendingQrUrl = data.verification_uri_complete;

			updateCountdown();
			countdownTimer = setInterval(updateCountdown, 1000);
			view = 'showing';
			pollTimer = setTimeout(() => poll(myFlowId), pollingInterval * 1000);
		});
	}

	function loadAppName() {
		request('GET', '/api/application-configuration', null, (status, data) => {
			if (status >= 200 && status < 300 && Array.isArray(data)) {
				const entry = data.find((d: any) => d.key === 'appName' && d.value);
				if (entry) {
					appName = entry.value;
					document.title = `Sign In - ${entry.value}`;
				}
			}
		});
	}

	const codeSplitIndex = $derived(Math.floor(userCodeChars.length / 2));
	// aria-label mirrors the visible code (with dash separator) so WCAG 2.5.3 (Label in Name) holds.
	const userCodeAriaLabel = $derived(
		userCodeChars.length
			? userCodeChars.slice(0, codeSplitIndex).join('') +
					'-' +
					userCodeChars.slice(codeSplitIndex).join('')
			: ''
	);

	onMount(() => {
		redirectUrl = getRedirectParam();
		loadAppName();
		startFlow();
	});

	onDestroy(() => cleanup());
</script>

<main class="card">
	{#if view === 'loading'}
		<div class="state" tabindex="-1" role="status" bind:this={stateEl}>
			<div class="spinner" aria-hidden="true"></div>
			<p class="muted">Preparing sign-in...</p>
		</div>
	{:else if view === 'showing'}
		<div class="state" tabindex="-1" bind:this={stateEl}>
			<h1>{appName}</h1>
			<p class="subtitle">Sign in with another device</p>
			<p class="muted">
				Scan the QR code with your phone or enter the code below on a device where you are already
				signed in.
			</p>

			<div id="qr-container">
				<canvas id="qr-canvas" aria-label="QR code for sign-in"></canvas>
			</div>

			<div class="code-section">
				<p class="code-label">Your sign-in code</p>
				<div class="code-boxes" role="group" aria-label={userCodeAriaLabel}>
					{#each userCodeChars as char, i}
						{#if i === codeSplitIndex && userCodeChars.length > 1}
							<span class="code-separator" aria-hidden="true">&ndash;</span>
						{/if}
						<span class="code-box" aria-hidden="true">{char}</span>
					{/each}
				</div>
			</div>

			<p
				class="muted small"
				role="timer"
				aria-label="Sign-in code expiry countdown"
				aria-live="off"
			>
				{countdown}
			</p>
		</div>
	{:else if view === 'authorized'}
		<div class="state" tabindex="-1" aria-live="assertive" aria-atomic="true" bind:this={stateEl}>
			<div class="status-icon status-success">
				<svg
					aria-hidden="true"
					width="32"
					height="32"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2.5"
					stroke-linecap="round"
					stroke-linejoin="round"
				>
					<polyline points="20 6 9 17 4 12"></polyline>
				</svg>
			</div>
			<h1>Authorized</h1>
			<p class="muted">You are signed in. Redirecting...</p>
		</div>
	{:else if view === 'expired'}
		<div class="state" tabindex="-1" aria-live="polite" aria-atomic="true" bind:this={stateEl}>
			<div class="status-icon status-warning">
				<svg
					aria-hidden="true"
					width="32"
					height="32"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2.5"
					stroke-linecap="round"
					stroke-linejoin="round"
				>
					<circle cx="12" cy="12" r="10"></circle>
					<polyline points="12 6 12 12 16 14"></polyline>
				</svg>
			</div>
			<h1>Code Expired</h1>
			<p class="muted">The sign-in code has expired. Please generate a new one.</p>
			<button type="button" onclick={startFlow}>Generate New Code</button>
		</div>
	{:else if view === 'error'}
		<div class="state" tabindex="-1" aria-live="assertive" aria-atomic="true" bind:this={stateEl}>
			<div class="status-icon status-error">
				<svg
					aria-hidden="true"
					width="32"
					height="32"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2.5"
					stroke-linecap="round"
					stroke-linejoin="round"
				>
					<circle cx="12" cy="12" r="10"></circle>
					<line x1="15" y1="9" x2="9" y2="15"></line>
					<line x1="9" y1="9" x2="15" y2="15"></line>
				</svg>
			</div>
			<h1>Something went wrong</h1>
			<p class="muted">{errorMessage}</p>
			<button type="button" onclick={startFlow}>Try Again</button>
		</div>
	{/if}
</main>
