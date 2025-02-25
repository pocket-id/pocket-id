<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import OIDCService from '$lib/services/oidc-service';
	import type { OidcClient } from '$lib/types/oidc.type';
	import userStore from '$lib/stores/user-store';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { toast } from 'svelte-sonner';

	let { data } = $props<{
		data: { code: string | null; client?: OidcClient; mode: 'verify' | 'authorize' };
	}>();

	const oidcService = new OIDCService();
	let userCode = $state(data.code || '');
	let isLoading = $state(false);

	onMount(() => {
		if (data.code && data.mode === 'verify') {
			verifyCode();
		}
	});

	async function verifyCode() {
		if (!userCode) return;

		isLoading = true;
		try {
			await oidcService.verifyDeviceCode(userCode);
			toast.success('Device successfully authorized');
			goto('/');
		} catch (e) {
			axiosErrorToast(e);
		} finally {
			isLoading = false;
		}
	}

	async function authorize() {
		if (!data.client) return;

		isLoading = true;
		try {
			const deviceAuth = await oidcService.deviceAuthorize(data.client.id, 'openid profile email');
			userCode = deviceAuth.userCode;
			await verifyCode();
		} catch (e) {
			axiosErrorToast(e);
		} finally {
			isLoading = false;
		}
	}
</script>

<div class="container mx-auto max-w-lg py-8">
	{#if data.mode === 'authorize'}
		<h1 class="mb-8 text-2xl font-bold">Authorize Device</h1>
		<div class="mb-4">
			<p>Allow {data.client?.name} to access your account?</p>
		</div>
		<button on:click={authorize} disabled={isLoading} class="bg-primary rounded px-4 py-2">
			{isLoading ? 'Authorizing...' : 'Authorize'}
		</button>
	{:else}
		<h1 class="mb-8 text-2xl font-bold">Device Verification</h1>
		<div class="mb-4">
			<label for="userCode" class="mb-2 block font-medium">Enter Code</label>
			<input
				type="text"
				id="userCode"
				bind:value={userCode}
				class="w-full rounded border p-2"
				placeholder="Enter the code shown on your device"
			/>
		</div>
		<button on:click={verifyCode} disabled={isLoading} class="bg-primary rounded px-4 py-2">
			{isLoading ? 'Verifying...' : 'Verify'}
		</button>
	{/if}
</div>
