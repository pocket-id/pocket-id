<script lang="ts">
	import { Checkbox as CheckboxPrimitive } from 'bits-ui';
	import Check from 'lucide-svelte/icons/check';
	import Minus from 'lucide-svelte/icons/minus';
	import { cn } from '$lib/utils/style.js';

	type $$Props = CheckboxPrimitive.Props;
	type $$Events = CheckboxPrimitive.Events;

	interface Props {
		class?: $$Props['class'];
		checked?: $$Props['checked'];
		[key: string]: any
	}

	let { class: className = undefined, checked = $bindable(false), ...rest }: Props = $props();
	
</script>

<CheckboxPrimitive.Root
	class={cn(
		'peer border-primary ring-offset-background focus-visible:ring-ring data-[state=checked]:bg-primary data-[state=checked]:text-primary-foreground box-content h-4 w-4 shrink-0 rounded-sm border focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50 data-[disabled=true]:cursor-not-allowed data-[disabled=true]:opacity-50',
		className
	)}
	bind:checked
	{...rest}
	on:click
>
	<CheckboxPrimitive.Indicator
		class={cn('flex h-4 w-4 items-center justify-center text-current')}
		
		
	>
		{#snippet children({ isChecked, isIndeterminate })}
				{#if isChecked}
				<Check class="h-3.5 w-3.5" />
			{:else if isIndeterminate}
				<Minus class="h-3.5 w-3.5" />
			{/if}
					{/snippet}
		</CheckboxPrimitive.Indicator>
</CheckboxPrimitive.Root>
