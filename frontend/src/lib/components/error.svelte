<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import { LucideXCircle } from 'lucide-svelte';

	let { message, showButton = true }: { message: string; showButton?: boolean } = $props();
</script>

<div class="mt-[20%] flex flex-col items-center">
	<LucideXCircle class="h-12 w-12 text-muted-foreground" />
	<h1 class="mt-3 text-2xl font-semibold">{m.something_went_wrong()}</h1>
	<p class="text-muted-foreground">{message}</p>
	{#if showButton}
		<Button size="sm" class="mt-5" href="/">{m.go_back_to_home()}</Button>
	{/if}
</div>
