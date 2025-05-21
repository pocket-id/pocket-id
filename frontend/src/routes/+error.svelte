<script lang="ts">
	import { page } from '$app/state';
	import Error from '$lib/components/error.svelte';
</script>

<Error message={page.error!.message} />
