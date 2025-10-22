<script lang="ts">
	import { page } from '$app/state';
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import Logo from '$lib/components/logo.svelte';
	import { m } from '$lib/paraglide/messages';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import { LucideMail, LucideRectangleEllipsis } from '@lucide/svelte';
	import LoginMethodCard from '../components/login-method-card.svelte';

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
</script>

<svelte:head>
	<title>{m.sign_in()}</title>
</svelte:head>

<SignInWrapper>
	<div class="flex h-full flex-col justify-center">
		<div class="bg-muted mx-auto rounded-2xl p-3">
			<Logo class="size-10" />
		</div>
		<h1 class="font-playfair mt-5 text-3xl font-bold sm:text-4xl">{m.alternative_sign_in()}</h1>
		<p class="text-muted-foreground mt-3">
			{m.if_you_do_not_have_access_to_your_passkey_you_can_sign_in_using_one_of_the_following_methods()}
		</p>
		<div class="mt-5 flex flex-col gap-3">
			{#each methods as method}
				<LoginMethodCard
					icon={method.icon}
					title={method.title}
					description={method.description}
					href={method.href}
					searchParams={page.url.search}
				/>
			{/each}
		</div>

		<a class="text-muted-foreground mt-5 text-xs" href={'/login' + page.url.search}
			>{m.use_your_passkey_instead()}</a
		>
	</div>
</SignInWrapper>
