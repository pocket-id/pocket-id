<script lang="ts" module>
	import type { FormPath, SuperForm } from 'sveltekit-superforms';
	type T = Record<string, unknown>;
	type U = FormPath<T>;
</script>

<script lang="ts" generics="T extends Record<string, unknown>, U extends FormPath<T>">
	import * as FormPrimitive from 'formsnap';
	import { cn } from '$lib/utils/style.js';

	type $$Props = FormPrimitive.FieldsetProps<T, U>;


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

<FormPrimitive.Fieldset
	{form}
	{name}
	
	
	
	
	class={cn('space-y-2', className)}
>
	{#snippet children({ constraints, errors, tainted, value })}
		{@render children_render?.({ constraints, errors, tainted, value, })}
	{/snippet}
</FormPrimitive.Fieldset>
