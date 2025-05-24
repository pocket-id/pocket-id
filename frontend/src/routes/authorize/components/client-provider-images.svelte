<script lang="ts">
	import Logo from '$lib/components/logo.svelte';
	import CheckmarkAnimated from '$lib/icons/checkmark-animated.svelte';
	import ConnectArrow from '$lib/icons/connect-arrow.svelte';
	import CrossAnimated from '$lib/icons/cross-animated.svelte';
	import { m } from '$lib/paraglide/messages';
	import type { OidcClientMetaData } from '$lib/types/oidc.type';

	const {
		success,
		error,
		client
	}: {
		success: boolean;
		error: boolean;
		client: OidcClientMetaData;
	} = $props();

	let animationDone = $state(false);

	$effect(() => {
		if (success || error) {
			setTimeout(() => {
				animationDone = true;
			}, 500);
		} else {
			animationDone = false;
		}
	});
</script>

<div class="flex justify-center gap-3">
	<div
		class=" bg-muted transition-translate rounded-2xl p-3 duration-500 ease-in {success || error
			? 'translate-x-[108px]'
			: ''}"
	>
		<Logo class="size-10" />
	</div>

	<ConnectArrow
		class="h-w-32 w-32 transition-opacity duration-500 {success || error
			? 'opacity-0'
			: 'opacity-100 delay-300'}"
	/>
	<div
		class="rounded-2xl p-3 [transition:translate_500ms_ease-in,background-color_200ms] {success ||
		error
			? '-translate-x-[108px]'
			: ''} {animationDone ? (success ? 'bg-green-200' : 'bg-red-200') : 'bg-muted'}"
	>
		{#if animationDone && success}
			<div class="flex size-10 items-center justify-center">
				<CheckmarkAnimated class="size-7" />
			</div>
		{:else if animationDone && error}
			<div class="flex size-10 items-center justify-center">
				<CrossAnimated class="size-5" />
			</div>
		{:else if client.hasLogo}
			<img
				class="size-10"
				src="/api/oidc/clients/{client.id}/logo"
				draggable={false}
				alt={m.client_logo()}
			/>
		{:else}
			<div class="flex size-10 items-center justify-center text-3xl font-bold">
				{client.name.charAt(0).toUpperCase()}
			</div>
		{/if}
	</div>
</div>
