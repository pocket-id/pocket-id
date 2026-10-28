<script lang="ts">
	import { page } from '$app/state';
	import Error from '#lib/components/error.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import type { ErrorProps } from './$types';

	let { error }: ErrorProps = $props();
</script>

<!-- Errors under /settings render inside the shell, so the navigation stays -->
<!-- Without a matched route the message is SvelteKit's untranslated default -->
<Error message={page.route.id ? error.message : m.page_not_found()} />
