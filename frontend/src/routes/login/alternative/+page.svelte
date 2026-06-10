<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import Logo from '$lib/components/logo.svelte';
	import QrLoginFlow from '$lib/components/qr-login-flow.svelte';
	import * as Item from '$lib/components/ui/item/index.js';
	import { m } from '$lib/paraglide/messages';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import { isSafeRedirect } from '$lib/utils/redirection-util';
	import { LucideChevronRight, LucideMail, LucideRectangleEllipsis } from '@lucide/svelte';

	const methods = [
		{
			icon: LucideRectangleEllipsis,
			title: m.login_code(),
			description: m.enter_a_login_code_to_sign_in(),
			href: '/login/alternative/code'
		}
	];

	if ($appConfigStore.emailOneTimeAccessAsUnauthenticatedEnabled) {
		methods.push({
			icon: LucideMail,
			title: m.email_login(),
			description: m.request_a_login_code_via_email(),
			href: '/login/alternative/email'
		});
	}

	function onAuthorized() {
		const redirect = page.url.searchParams.get('redirect');
		if (redirect && isSafeRedirect(redirect)) {
			goto(redirect);
		} else {
			goto('/');
		}
	}

	function getPasskeyLoginHref() {
		const params = new URLSearchParams(page.url.search);
		params.set('method', 'passkey');
		return '/login?' + params.toString();
	}
</script>

<svelte:head>
	<title>{m.sign_in()}</title>
</svelte:head>

<SignInWrapper>
	<div class="flex h-full flex-col justify-center">
		<div class="bg-muted mx-auto rounded-2xl p-3">
			<Logo class="size-10" />
		</div>
		<h1 class="font-gloock mt-5 text-3xl font-bold sm:text-4xl">{m.alternative_sign_in()}</h1>
		<p class="text-muted-foreground mt-3">
			{m.if_you_do_not_have_access_to_your_passkey_you_can_sign_in_using_one_of_the_following_methods()}
		</p>

		{#if $appConfigStore.qrLoginEnabled}
			<div class="mt-5">
				<QrLoginFlow onauthorized={onAuthorized} />
			</div>
		{/if}

		<div class={$appConfigStore.qrLoginEnabled ? 'border-border mt-6 border-t pt-4' : 'mt-5'}>
			{#if $appConfigStore.qrLoginEnabled}
				<p class="text-muted-foreground mb-3 text-sm">{m.other_methods()}</p>
			{/if}
			<Item.Group class="gap-3">
				{#each methods as method}
					<Item.Root variant="outline" class="gap-5">
						{#snippet child({ props })}
							<a href={method.href + page.url.search} {...props}>
								<Item.Media class="text-primary !self-center !translate-y-0">
									<method.icon class="size-7" />
								</Item.Media>
								<Item.Content class="text-start">
									<Item.Title class="text-lg font-semibold">{method.title}</Item.Title>
									<Item.Description>{method.description}</Item.Description>
								</Item.Content>
								<Item.Actions>
									<LucideChevronRight class="size-5" />
								</Item.Actions>
							</a>
						{/snippet}
					</Item.Root>
				{/each}
			</Item.Group>
		</div>

		<a class="text-muted-foreground mt-5 text-xs" href={getPasskeyLoginHref()}
			>{m.use_your_passkey_instead()}</a
		>
	</div>
</SignInWrapper>
