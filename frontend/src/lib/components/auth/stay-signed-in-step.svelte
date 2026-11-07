<script lang="ts">
	import AuthHeading from '#lib/components/auth/auth-heading.svelte';
	import AuthTile from '#lib/components/auth/auth-tile.svelte';
	import { authPrimaryClass, authQuietClass } from '#lib/components/auth/auth-util.ts';
	import { Button } from '@pocket-id/ui/button';
	import { Checkbox } from '@pocket-id/ui/checkbox';
	import { m } from '#lib/paraglide/messages.js';
	import SessionService from '#lib/services/session-service.ts';
	import { keepSignedIn } from '#lib/stores/keep-signed-in-store.svelte.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { cssMs } from '#lib/utils/style.ts';
	import { onDestroy } from 'svelte';
	import { fade } from 'svelte/transition';

	const sessionService = new SessionService();

	let isLoading = $state(false);
	let remember = $state(true);

	onDestroy(keepSignedIn.dismiss);

	async function stay() {
		isLoading = true;
		try {
			await sessionService.rememberCurrent();
			keepSignedIn.choose(true, remember);
		} catch (e) {
			axiosErrorToast(e);
		}
		isLoading = false;
	}
</script>

<div class="flex flex-col" in:fade={{ duration: cssMs('--dur-base') }}>
	<AuthTile />
	<AuthHeading title={m.auth_stay_signed_in_question()}>
		{m.auth_stay_signed_in_description()}
	</AuthHeading>
	<label class="text-body text-fg-muted mb-6 flex cursor-pointer items-center gap-3">
		<Checkbox bind:checked={remember} />
		{m.auth_dont_ask_again_in_this_browser()}
	</label>
	<div class="flex flex-col items-center gap-3">
		<Button class={authPrimaryClass} {isLoading} onclick={stay}>
			{m.auth_stay_signed_in()}
		</Button>
		<Button
			variant="link"
			class={authQuietClass}
			disabled={isLoading}
			onclick={() => keepSignedIn.choose(false, remember)}
		>
			{m.auth_dont_stay_signed_in()}
		</Button>
	</div>
</div>
