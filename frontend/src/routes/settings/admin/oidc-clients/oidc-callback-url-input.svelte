<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import UrlListInput from '$lib/components/form/url-list-input.svelte';
	import * as Field from '$lib/components/ui/field';
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		label,
		description,
		callbackURLs = $bindable(),
		error = $bindable(null),
		...restProps
	}: HTMLAttributes<HTMLDivElement> & {
		label: string;
		description: string;
		callbackURLs: string[];
		error?: string | null;
		children?: Snippet;
	} = $props();
</script>

<div {...restProps}>
	<FormInput {label} {description}>
		<UrlListInput bind:urls={callbackURLs} {error} testIdPrefix="callback-url" />
	</FormInput>
	{#if error}
		<Field.Error>{error}</Field.Error>
	{/if}
</div>
