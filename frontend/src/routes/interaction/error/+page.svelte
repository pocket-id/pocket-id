<script lang="ts">
	import SignInWrapper from '#lib/components/login-wrapper.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import userStore from '#lib/stores/user-store.ts';
	import { isEmailVerificationPending } from '#lib/utils/email-verification-util.ts';
	import ClientProviderImages from '../../authorize/components/client-provider-images.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	let { error } = data;
</script>

<svelte:head>
	<title>{m.error()}</title>
</svelte:head>

<SignInWrapper>
	<ClientProviderImages error success={false} />
	<h1 class="font-display mt-5 text-3xl font-bold sm:text-4xl">
		{m.error()}
	</h1>
	<p class="text-muted-foreground mt-2 mb-10">
		{error}
	</p>

	<div class="flex w-full flex-col items-center gap-3">
		{#if isEmailVerificationPending($userStore, $appConfigStore)}
			<div class="flex w-full max-w-md flex-row-reverse gap-2">
				<Button class="flex-1" href="/settings/account">
					{m.verify_email_address()}
				</Button>
				<Button class="flex-1" variant="secondary" href={document.referrer || '/'}>
					{m.go_back()}
				</Button>
			</div>
		{:else}
			<Button class="w-full sm:w-[50%]" variant="secondary" href={document.referrer || '/'}>
				{m.go_back()}
			</Button>
		{/if}
	</div>
</SignInWrapper>
