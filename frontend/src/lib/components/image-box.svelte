<script lang="ts">
	import { cn } from '$lib/utils/style';
	import type { HTMLImgAttributes } from 'svelte/elements';

	let props: HTMLImgAttributes & {} = $props();
</script>

<div class={cn('bg-muted flex size-24 items-center justify-center rounded-2xl p-3', props.class)}>
	<img class='object-contain' {...props} />
</div>
