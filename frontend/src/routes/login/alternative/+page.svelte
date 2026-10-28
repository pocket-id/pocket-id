<script lang="ts">
	import { page } from '$app/state';
	import AuthHeading from '#lib/components/auth/auth-heading.svelte';
	import AuthTile from '#lib/components/auth/auth-tile.svelte';
	import { authQuietClass, emailedLoginCode } from '#lib/components/auth/auth-util.ts';
	import SignInWrapper from '#lib/components/login-wrapper.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import { LucideChevronRight, LucideKeyboard, LucideMail, LucideSmartphone } from '@lucide/svelte';

	const emailCodes = $appConfigStore.emailOneTimeAccessAsUnauthenticatedEnabled;

	// Choosing a method again starts over, so Enter a login code no longer refers to an earlier email
	emailedLoginCode.sentTo = '';

	const methods = [
		{
			icon: LucideSmartphone,
			title: m.auth_use_another_device(),
			description: m.auth_use_another_device_description(),
			href: '/login/alternative/device'
		},
		{
			icon: LucideKeyboard,
			title: m.auth_enter_a_login_code(),
			description: emailCodes
				? m.auth_enter_a_login_code_description()
				: m.auth_enter_an_admin_login_code_description(),
			href: '/login/alternative/code'
		},
		...(emailCodes
			? [
					{
						icon: LucideMail,
						title: m.auth_email_me_a_login_code(),
						description: m.auth_email_me_a_login_code_description(),
						href: '/login/alternative/email'
					}
				]
			: [])
	];
</script>

<svelte:head>
	<title>{m.auth_other_ways_to_sign_in()}</title>
</svelte:head>

<SignInWrapper back={'/login' + page.url.search}>
	<AuthTile />
	<AuthHeading title={m.auth_other_ways_to_sign_in()}>
		{m.auth_other_ways_to_sign_in_description()}
	</AuthHeading>
	<ul class="-mx-3 mb-6 flex flex-col gap-0.5">
		{#each methods as method (method.href)}
			<li>
				<a
					href={method.href + page.url.search}
					class="group/method before:bg-fill-row-hover transition-motion focus-visible:outline-ring relative isolate flex min-h-16 items-center gap-3 rounded-md p-3 before:absolute before:inset-0 before:-z-1 before:rounded-[inherit] before:opacity-0 before:transition-opacity before:duration-(--dur-fast) hover:before:opacity-100 focus-visible:outline-2 focus-visible:outline-offset-2 active:scale-(--press-scale)"
				>
					<span class="bg-fill-muted grid size-8 shrink-0 place-items-center rounded-[8px]">
						<method.icon class="size-4" />
					</span>
					<span class="flex min-w-0 flex-1 flex-col gap-0.5">
						<span class="text-heading font-medium">{method.title}</span>
						<span class="text-control text-fg-muted">{method.description}</span>
					</span>
					<LucideChevronRight
						class="text-fg-muted size-4 shrink-0 transition-transform duration-(--dur-fast) group-hover/method:translate-x-(--nudge)"
					/>
				</a>
			</li>
		{/each}
	</ul>
	<Button variant="link" class={authQuietClass} href={'/login' + page.url.search}>
		{m.auth_use_your_passkey_instead()}
	</Button>
</SignInWrapper>
