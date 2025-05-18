<script lang="ts">
	import * as FormPrimitive from 'formsnap';
	import { cn } from '$lib/utils/style.js';

	type $$Props = FormPrimitive.FieldErrorsProps & {
		errorClasses?: string | undefined | null;
	};

	
	interface Props {
		class?: $$Props['class'];
		errorClasses?: $$Props['class'];
		children?: import('svelte').Snippet<[any]>;
		[key: string]: any
	}

	let { class: className = undefined, errorClasses = undefined, children, ...rest }: Props = $props();

	const children_render = $derived(children);
</script>

<FormPrimitive.FieldErrors
	class={cn('text-destructive text-sm font-medium', className)}
	{...rest}
	
	
	
>
	{#snippet children({ errors, fieldErrorsAttrs, errorAttrs })}
		{#if children_render}{@render children_render({ errors, fieldErrorsAttrs, errorAttrs, })}{:else}
			{#each errors as error}
				<div {...errorAttrs} class={cn(errorClasses)}>{error}</div>
			{/each}
		{/if}
	{/snippet}
</FormPrimitive.FieldErrors>
