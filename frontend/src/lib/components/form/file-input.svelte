<script lang="ts">
	import { cn } from '#lib/utils/style.ts';
	import type { HTMLInputAttributes } from 'svelte/elements';
	import type { VariantProps } from 'tailwind-variants';
	import type { buttonVariants } from '#lib/components/ui/button/index.ts';
	import { m } from '#lib/paraglide/messages.js';

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
		{m.select_file()}
	{/if}
</button>
<input {id} {...restProps} type="file" class="hidden" />
