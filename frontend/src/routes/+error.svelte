<script lang="ts">
	import { page } from '$app/state';
	import AuthHeading from '#lib/components/auth/auth-heading.svelte';
	import AuthTile from '#lib/components/auth/auth-tile.svelte';
	import { authPrimaryClass } from '#lib/components/auth/auth-util.ts';
	import SignInWrapper from '#lib/components/login-wrapper.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import type { ErrorProps } from './$types';

	let { error }: ErrorProps = $props();

	const title = $derived(page.status === 404 ? m.page_not_found_title() : m.something_went_wrong());
</script>

<svelte:head>
	<title>{title}</title>
</svelte:head>

<SignInWrapper>
	<AuthTile />
	<!-- Without a matched route the message is SvelteKit's untranslated default -->
	<AuthHeading {title}>
		{page.route.id ? error.message : m.page_not_found()}
	</AuthHeading>
	<Button class={authPrimaryClass} href="/">{m.go_back_to_home()}</Button>
</SignInWrapper>
