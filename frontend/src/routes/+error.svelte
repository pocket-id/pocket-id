<script lang="ts">
	import Error from '#lib/components/error.svelte';

	let { error } = $props();
</script>

<Error message={error.message} />
