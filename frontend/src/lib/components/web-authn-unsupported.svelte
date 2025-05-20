<script>
	import { m } from '$lib/paraglide/messages';
	import Logo from './logo.svelte';
</script>

<div class="flex flex-col justify-center">
	<div class="bg-muted mx-auto rounded-2xl p-3">
		<Logo class="size-10" />
	</div>
	<p class="font-playfair mt-5 text-3xl font-bold sm:text-4xl">{m.browser_unsupported()}</p>
	<p class="text-muted-foreground mt-3">
		{m.this_browser_does_not_support_passkeys()}
	</p>
</div>
