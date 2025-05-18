<script lang="ts" module>
	import type { FormPath, SuperForm } from 'sveltekit-superforms';
	type T = Record<string, unknown>;
	type U = FormPath<T>;
</script>

<script lang="ts" generics="T extends Record<string, unknown>, U extends FormPath<T>">
	import type { HTMLAttributes } from 'svelte/elements';
	import * as FormPrimitive from 'formsnap';
	import { cn } from '$lib/utils/style.js';

	type $$Props = FormPrimitive.FieldProps<T, U> & HTMLAttributes<HTMLElement>;


	interface Props {
		form: SuperForm<T>;
		name: U;
		class?: $$Props['class'];
		children?: import('svelte').Snippet<[any]>;
	}

	let {
		form,
		name,
		class: className = undefined,
		children
	}: Props = $props();
	

	const children_render = $derived(children);
</script>

<FormPrimitive.Field {form} {name}    >
	{#snippet children({ constraints, errors, tainted, value })}
		<div class={cn('space-y-2', className)}>
			{@render children_render?.({ constraints, errors, tainted, value, })}
		</div>
	{/snippet}
</FormPrimitive.Field>
