<script lang="ts">
	export let icon: ConstructorOfATypedSvelteComponent;
	export let name: string;
	export let description: string;
</script>

<div class="flex items-center">
	<div class="mr-5 rounded-lg bg-muted p-2"><svelte:component this={icon} /></div>
	<div class="text-start">
		<h3 class="font-semibold">{name}</h3>
		<p class="text-sm text-muted-foreground">{description}</p>
	</div>
</div>
