<script lang="ts">
	import * as Button from '$lib/components/ui/button/index.js';
	interface Props {
		children?: import('svelte').Snippet;
		[key: string]: any
	}

	let { children, ...rest }: Props = $props();

	type $$Props = Button.Props;
	type $$Events = Button.Events;
</script>

<Button.Root type="submit" on:click on:keydown {...rest}>
	{@render children?.()}
</Button.Root>
