<script lang="ts">
	import { page } from '$app/state';
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import Logo from '$lib/components/logo.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import { LucideChevronRight, LucideMail, LucideRectangleEllipsis } from 'lucide-svelte';

	const methods = [
		{
			icon: LucideRectangleEllipsis,
			title: 'Login Code',
			description: 'Enter a login code to sign in.',
			href: '/login/alternative/code'
		}
	];

	if ($appConfigStore.emailOneTimeAccessEnabled) {
		methods.push({
			icon: LucideMail,
			title: 'Email Login',
			description: 'Request a login code via email.',
			href: '/login/alternative/email'
		});
	}
</script>

<svelte:head>
	<title>Sign In</title>
</svelte:head>

<SignInWrapper>
	<div class="flex h-full flex-col justify-center">
		<div class="bg-muted mx-auto rounded-2xl p-3">
			<Logo class="h-10 w-10" />
		</div>
		<h1 class="font-playfair mt-5 text-3xl font-bold sm:text-4xl">Alternative Sign In</h1>
		<p class="text-muted-foreground mt-3">
			If you dont't have access to your passkey, you can sign in using one of the following methods.
		</p>
		<div class="mt-5 flex flex-col gap-3">
			{#each methods as method}
				<a href={method.href + page.url.search}>
					<Card.Root>
						<Card.Content class="flex items-center justify-between p-4">
							<div class="flex gap-3">
								<method.icon class="text-primary h-7 w-7" />
								<div class="text-start">
									<h3 class="text-lg font-semibold">{method.title}</h3>
									<p class="text-muted-foreground text-sm">{method.description}</p>
								</div>
							</div>
							<Button variant="ghost"><LucideChevronRight class="h-5 w-5" /></Button>
						</Card.Content>
					</Card.Root>
				</a>
			{/each}
		</div>

		<a class="text-muted-foreground mt-5 text-xs" href={'/login' + page.url.search}
			>Use your passkey instead?</a
		>
	</div>
</SignInWrapper>
