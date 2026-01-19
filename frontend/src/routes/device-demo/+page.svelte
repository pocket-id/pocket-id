<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { onMount } from 'svelte';
	import QRCode from 'qrcode';

	let clientId = $state('');
	let clientSecret = $state('');
	let scope = $state('openid profile email');
	
	let deviceCode = $state('');
	let userCode = $state('');
	let verificationUri = $state('');
	let verificationUriComplete = $state('');
	let qrCodeUri = $state('');
	let expiresIn = $state(0);
	let interval = $state(5);
	
	let qrCodeDataUrl = $state('');
	let isLoading = $state(false);
	let error = $state('');
	let success = $state(false);
	let polling = $state(false);
	let pollingInterval: ReturnType<typeof setInterval> | null = null;

	async function initiateDeviceFlow() {
		isLoading = true;
		error = '';
		success = false;
		qrCodeDataUrl = '';
		
		try {
			const formData = new URLSearchParams();
			formData.append('client_id', clientId);
			formData.append('client_secret', clientSecret);
			formData.append('scope', scope);

			const response = await fetch('/api/oidc/device/authorize', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/x-www-form-urlencoded',
				},
				body: formData
			});

			if (!response.ok) {
				throw new Error(`Failed to initiate device flow: ${response.statusText}`);
			}

			const data = await response.json();
			deviceCode = data.device_code;
			userCode = data.user_code;
			verificationUri = data.verification_uri;
			verificationUriComplete = data.verification_uri_complete;
			qrCodeUri = data.qr_code_uri;
			expiresIn = data.expires_in;
			interval = data.interval;

			// Generate QR code from verification URI
			qrCodeDataUrl = await QRCode.toDataURL(verificationUriComplete, {
				width: 300,
				margin: 2,
				errorCorrectionLevel: 'M'
			});

			// Start polling for authorization
			startPolling();
		} catch (e: any) {
			error = e.message || 'Failed to initiate device flow';
		} finally {
			isLoading = false;
		}
	}

	function startPolling() {
		polling = true;
		pollingInterval = setInterval(async () => {
			try {
				const formData = new URLSearchParams();
				formData.append('grant_type', 'urn:ietf:params:oauth:grant-type:device_code');
				formData.append('device_code', deviceCode);
				formData.append('client_id', clientId);
				formData.append('client_secret', clientSecret);

				const response = await fetch('/api/oidc/token', {
					method: 'POST',
					headers: {
						'Content-Type': 'application/x-www-form-urlencoded',
					},
					body: formData
				});

				if (response.ok) {
					const tokens = await response.json();
					success = true;
					polling = false;
					if (pollingInterval) {
						clearInterval(pollingInterval);
						pollingInterval = null;
					}
					console.log('Received tokens:', tokens);
				} else {
					const errorData = await response.json();
					// authorization_pending and slow_down are expected during polling
					if (errorData.error !== 'authorization_pending' && errorData.error !== 'slow_down') {
						throw new Error(errorData.error || 'Token exchange failed');
					}
				}
			} catch (e: any) {
				error = e.message || 'Polling failed';
				polling = false;
				if (pollingInterval) {
					clearInterval(pollingInterval);
					pollingInterval = null;
				}
			}
		}, interval * 1000);
	}

	function reset() {
		deviceCode = '';
		userCode = '';
		verificationUri = '';
		verificationUriComplete = '';
		qrCodeUri = '';
		qrCodeDataUrl = '';
		success = false;
		error = '';
		polling = false;
		if (pollingInterval) {
			clearInterval(pollingInterval);
			pollingInterval = null;
		}
	}

	onMount(() => {
		return () => {
			if (pollingInterval) {
				clearInterval(pollingInterval);
			}
		};
	});
</script>

<svelte:head>
	<title>Device Flow QR Code Demo - Pocket ID</title>
</svelte:head>

<div class="container mx-auto max-w-4xl p-6">
	<div class="mb-8">
		<h1 class="text-4xl font-bold mb-2">OIDC Device Flow with QR Code</h1>
		<p class="text-muted-foreground">
			This demo shows how to implement OIDC device authorization flow with QR codes for remote devices like smart TVs, IoT devices, or kiosks.
		</p>
	</div>

	<div class="grid gap-6 md:grid-cols-2">
		<!-- Configuration Card -->
		<Card.Root>
			<Card.Header>
				<Card.Title>Configuration</Card.Title>
				<Card.Description>Enter your OIDC client credentials</Card.Description>
			</Card.Header>
			<Card.Content class="space-y-4">
				<div class="space-y-2">
					<Label for="clientId">Client ID</Label>
					<Input 
						id="clientId" 
						bind:value={clientId} 
						placeholder="Enter your client ID"
						disabled={polling || success}
					/>
				</div>
				<div class="space-y-2">
					<Label for="clientSecret">Client Secret</Label>
					<Input 
						id="clientSecret" 
						type="password"
						bind:value={clientSecret} 
						placeholder="Enter your client secret"
						disabled={polling || success}
					/>
				</div>
				<div class="space-y-2">
					<Label for="scope">Scope</Label>
					<Input 
						id="scope" 
						bind:value={scope} 
						placeholder="openid profile email"
						disabled={polling || success}
					/>
				</div>
				<Button 
					onclick={initiateDeviceFlow} 
					disabled={!clientId || !clientSecret || isLoading || polling || success}
					class="w-full"
				>
					{isLoading ? 'Initiating...' : 'Start Device Flow'}
				</Button>
				{#if success || error || polling}
					<Button 
						onclick={reset} 
						variant="outline"
						class="w-full"
					>
						Reset
					</Button>
				{/if}
			</Card.Content>
		</Card.Root>

		<!-- QR Code Display Card -->
		<Card.Root>
			<Card.Header>
				<Card.Title>QR Code</Card.Title>
				<Card.Description>Scan this with your mobile device</Card.Description>
			</Card.Header>
			<Card.Content class="space-y-4">
				{#if error}
					<div class="bg-destructive/10 text-destructive p-4 rounded-md">
						<p class="font-semibold">Error</p>
						<p class="text-sm">{error}</p>
					</div>
				{:else if success}
					<div class="bg-green-500/10 text-green-600 dark:text-green-400 p-4 rounded-md">
						<p class="font-semibold">✓ Authorization Complete!</p>
						<p class="text-sm">The device has been successfully authorized. Tokens have been received.</p>
					</div>
				{:else if qrCodeDataUrl}
					<div class="flex flex-col items-center space-y-4">
						<img src={qrCodeDataUrl} alt="QR Code" class="rounded-lg border" />
						{#if polling}
							<div class="text-center">
								<div class="animate-pulse text-blue-600 dark:text-blue-400 font-semibold">
									Waiting for authorization...
								</div>
								<p class="text-sm text-muted-foreground mt-1">
									Scan the QR code or enter the user code manually
								</p>
							</div>
						{/if}
					</div>
				{:else}
					<div class="flex items-center justify-center h-[300px] border-2 border-dashed rounded-lg">
						<p class="text-muted-foreground">QR code will appear here</p>
					</div>
				{/if}
			</Card.Content>
		</Card.Root>
	</div>

	<!-- Details Card -->
	{#if userCode}
		<Card.Root class="mt-6">
			<Card.Header>
				<Card.Title>Device Flow Details</Card.Title>
			</Card.Header>
			<Card.Content class="space-y-3">
				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<Label class="text-muted-foreground">User Code</Label>
						<p class="font-mono font-bold text-2xl">{userCode}</p>
					</div>
					<div>
						<Label class="text-muted-foreground">Expires In</Label>
						<p class="font-semibold">{expiresIn} seconds</p>
					</div>
				</div>
				<div>
					<Label class="text-muted-foreground">Verification URI</Label>
					<p class="font-mono text-sm break-all">
						<a href={verificationUriComplete} target="_blank" rel="noopener noreferrer" class="text-blue-600 hover:underline">
							{verificationUriComplete}
						</a>
					</p>
				</div>
				<div>
					<Label class="text-muted-foreground">QR Code Image URL</Label>
					<p class="font-mono text-sm break-all">
						<a href={qrCodeUri} target="_blank" rel="noopener noreferrer" class="text-blue-600 hover:underline">
							{qrCodeUri}
						</a>
					</p>
				</div>
			</Card.Content>
		</Card.Root>
	{/if}

	<!-- How it Works Card -->
	<Card.Root class="mt-6">
		<Card.Header>
			<Card.Title>How It Works</Card.Title>
		</Card.Header>
		<Card.Content>
			<ol class="list-decimal list-inside space-y-2 text-sm">
				<li>The remote device (TV, kiosk, etc.) requests authorization and displays a QR code</li>
				<li>User scans the QR code with their smartphone, opening the verification URL</li>
				<li>User authenticates on their phone using passkeys (biometrics, PIN, etc.)</li>
				<li>User confirms authorization for the remote device</li>
				<li>The remote device polls for tokens and receives them once authorized</li>
				<li>The remote device is now authenticated and can access protected resources</li>
			</ol>
			<div class="mt-4 p-4 bg-muted rounded-lg">
				<p class="text-sm font-semibold mb-2">Benefits:</p>
				<ul class="list-disc list-inside space-y-1 text-sm text-muted-foreground">
					<li>Passwordless authentication - no typing on remote devices</li>
					<li>Secure - credentials never leave the trusted mobile device</li>
					<li>User presence verification through biometrics</li>
					<li>Based on OIDC Device Authorization Grant (RFC 8628)</li>
				</ul>
			</div>
		</Card.Content>
	</Card.Root>
</div>
