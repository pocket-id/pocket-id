<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import OIDCService from '$lib/services/oidc-service';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { toast } from 'svelte-sonner';
	import * as Card from '$lib/components/ui/card';
	import { LucideMail, LucideUser, LucideUsers } from 'lucide-svelte';
	import ScopeItem from '../authorize/components/scope-item.svelte';
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import { Button } from '$lib/components/ui/button';
	import ClientProviderImages from '../authorize/components/client-provider-images.svelte';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import { slide } from 'svelte/transition';
	import LoginLogoErrorSuccessIndicator from '../login/components/login-logo-error-success-indicator.svelte';

	let { data } = $props<{
		data: { code: string | null; client?: OidcClient; mode: 'verify' | 'authorize' };
	}>();

	const oidcService = new OIDCService();
	let userCode = $state(data.code || '');
	let isLoading = $state(false);
	let deviceInfo = $state<{ clientId: string; clientName: string; scope: string } | null>(null);
	let showScopeConfirmation = $state(false);
	let success = $state(false);
	let error = $state(false);
	let authorizationComplete = $state(false);

	onMount(() => {
		if (data.code && data.mode === 'verify') {
			getDeviceCodeInfo(data.code);
		}
	});

	async function getDeviceCodeInfo(code: string) {
		isLoading = true;
		try {
			const info = await oidcService.getDeviceCodeInfo(code);
			deviceInfo = info;
			showScopeConfirmation = true;
		} catch (e) {
			error = true;
			axiosErrorToast(e);
		} finally {
			isLoading = false;
		}
	}

	async function verifyCode() {
		if (!userCode) return;

		if (!deviceInfo) {
			await getDeviceCodeInfo(userCode);
			return;
		}

		isLoading = true;
		try {
			await oidcService.verifyDeviceCode(userCode);
			success = true;
			setTimeout(() => {
				authorizationComplete = true;
				toast.success('Device successfully authorized');
			}, 1000);
		} catch (e) {
			error = true;
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
			error = true;
			axiosErrorToast(e);
		} finally {
			isLoading = false;
		}
	}
</script>

<svelte:head>
	<title>{data.mode === 'authorize' ? 'Authorize Device' : 'Verify Device Code'}</title>
</svelte:head>

<SignInWrapper showEmailOneTimeAccessButton={$appConfigStore.emailOneTimeAccessEnabled}>
	{#if data.mode === 'authorize' && data.client}
		<ClientProviderImages client={data.client} {success} {error} />
		<h1 class="font-playfair mt-5 text-3xl font-bold sm:text-4xl">Authorize Device</h1>
		<p class="text-muted-foreground mb-10 mt-2">
			Do you want to authorize <b>{data.client.name}</b> on your device?
		</p>
		<Card.Root class="mb-10 mt-6">
			<Card.Header class="pb-5">
				<p class="text-muted-foreground text-start">
					<b>{data.client.name}</b> wants to access the following information:
				</p>
			</Card.Header>
			<Card.Content data-testid="scopes">
				<div class="flex flex-col gap-3">
					<ScopeItem icon={LucideMail} name="Email" description="View your email address" />
					<ScopeItem icon={LucideUser} name="Profile" description="View your profile information" />
				</div>
			</Card.Content>
		</Card.Root>
		<div class="flex w-full justify-stretch gap-2">
			<Button onclick={() => goto('/')} class="w-full" variant="secondary">Cancel</Button>
			<Button class="w-full" disabled={isLoading} on:click={authorize}>
				{isLoading ? 'Authorizing...' : 'Authorize'}
			</Button>
		</div>
	{:else if showScopeConfirmation && deviceInfo}
		<ClientProviderImages
			client={{ id: deviceInfo.clientId, name: deviceInfo.clientName }}
			{success}
			{error}
		/>
		<h1 class="font-playfair mt-5 text-3xl font-bold sm:text-4xl">Authorize Device</h1>

		{#if authorizationComplete}
			<div transition:slide={{ duration: 300 }}>
				<Card.Root class="mb-10 mt-6">
					<Card.Header class="pb-5">
						<p class="text-success text-center font-semibold">Authorization Complete!</p>
					</Card.Header>
					<Card.Content>
						<div class="flex flex-col gap-3">
							<div class="mb-2 flex justify-center">
								<svg
									xmlns="http://www.w3.org/2000/svg"
									class="h-16 w-16 text-green-500"
									fill="none"
									viewBox="0 0 24 24"
									stroke="currentColor"
								>
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M5 13l4 4L19 7"
									/>
								</svg>
							</div>
							<p class="text-center">You may now return to your device.</p>
							<p class="text-muted-foreground mt-2 text-center text-sm">
								The application should continue automatically.
							</p>
						</div>
					</Card.Content>
				</Card.Root>
			</div>
			<Button class="w-full" variant="secondary" on:click={() => goto('/')}>
				Return to Dashboard
			</Button>
		{:else}
			<div transition:slide={{ duration: 300 }}>
				<Card.Root class="mb-10 mt-6">
					<Card.Header class="pb-5">
						<p class="text-muted-foreground text-start">
							<b>{deviceInfo.clientName}</b> wants to access the following information:
						</p>
					</Card.Header>
					<Card.Content data-testid="scopes">
						<div class="flex flex-col gap-3">
							{#if deviceInfo.scope.includes('email')}
								<ScopeItem icon={LucideMail} name="Email" description="View your email address" />
							{/if}
							{#if deviceInfo.scope.includes('profile')}
								<ScopeItem
									icon={LucideUser}
									name="Profile"
									description="View your profile information"
								/>
							{/if}
							{#if deviceInfo.scope.includes('groups')}
								<ScopeItem
									icon={LucideUsers}
									name="Groups"
									description="View the groups you are a member of"
								/>
							{/if}
						</div>
					</Card.Content>
				</Card.Root>
			</div>
			<div class="flex w-full justify-stretch gap-2">
				<Button onclick={() => goto('/')} class="w-full" variant="secondary">Cancel</Button>
				<Button class="w-full" disabled={isLoading} on:click={verifyCode}>
					{isLoading ? 'Authorizing...' : 'Authorize'}
				</Button>
			</div>
		{/if}
	{:else}
		<div class="flex justify-center">
			<LoginLogoErrorSuccessIndicator error={!!error} />
		</div>
		<h1 class="font-playfair text-3xl font-bold sm:text-4xl">Device Verification</h1>
		<p class="text-muted-foreground mb-10 mt-2">
			Enter the code shown on your device to authorize access.
		</p>
		<div class="mb-4 w-full">
			<input
				type="text"
				id="userCode"
				bind:value={userCode}
				class="w-full rounded border p-2 text-center text-lg tracking-widest"
				placeholder="Enter code"
			/>
		</div>
		<div class="flex w-full justify-stretch gap-2">
			<Button onclick={() => goto('/')} class="w-full" variant="secondary">Cancel</Button>
			<Button class="w-full" disabled={isLoading} on:click={verifyCode}>
				{isLoading ? 'Verifying...' : 'Verify'}
			</Button>
		</div>
	{/if}
</SignInWrapper>
