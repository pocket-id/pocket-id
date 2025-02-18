<script lang="ts">
	import { cn } from '$lib/utils/style';
	import type { HTMLInputAttributes } from 'svelte/elements';
	import type { VariantProps } from 'tailwind-variants';
	import type { buttonVariants } from '$lib/components/ui/button';

	let {
		id,
		...restProps
	}: HTMLInputAttributes & {
		id: string;
		variant?: VariantProps<typeof buttonVariants>['variant'];
	} = $props();
</script>

<button
	type="button"
	onclick={() => document.getElementById(id)?.click()}
	class={cn(restProps.class)}
>
	{#if restProps.children}
		{@render restProps.children()}
	{:else}
		Select File
	{/if}
</button>
<input {id} {...restProps} type="file" class="hidden" />
