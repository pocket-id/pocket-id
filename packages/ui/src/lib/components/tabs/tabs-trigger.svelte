<script lang="ts">
	import { Tabs as TabsPrimitive } from 'bits-ui';
	import { cn } from '#lib/utils/style.js';

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: TabsPrimitive.TriggerProps = $props();
</script>

<TabsPrimitive.Trigger
	bind:ref
	data-slot="tabs-trigger"
	class={cn(
		"group/tabs-trigger text-control text-fg-muted hover:text-fg focus-visible:outline-ring data-[state=active]:text-fg relative inline-flex shrink-0 items-center justify-center whitespace-nowrap transition-colors duration-(--dur-fast) focus-visible:outline-2 disabled:pointer-events-none disabled:opacity-45 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-3.5",
		'group-data-[variant=default]/tabs-list:h-(--h-sm) group-data-[variant=default]/tabs-list:rounded-sm group-data-[variant=default]/tabs-list:px-3 group-data-[variant=default]/tabs-list:focus-visible:outline-offset-2 group-data-[variant=default]/tabs-list:data-[state=active]:font-medium',
		'group-data-[variant=line]/tabs-list:h-11 group-data-[variant=line]/tabs-list:rounded-xs group-data-[variant=line]/tabs-list:px-0.5 group-data-[variant=line]/tabs-list:font-medium group-data-[variant=line]/tabs-list:focus-visible:-outline-offset-2',
		'group-data-[orientation=vertical]/tabs:w-full group-data-[orientation=vertical]/tabs:justify-start',
		className
	)}
	{...restProps}
>
	<!-- The active weight is reserved in the same grid cell so selecting a tab never moves its neighbours -->
	<span class="grid">
		<span
			aria-hidden="true"
			class="invisible col-start-1 row-start-1 inline-flex items-center gap-1.5 font-medium"
		>
			{@render children?.()}
		</span>
		<span class="col-start-1 row-start-1 inline-flex items-center gap-1.5">
			{@render children?.()}
		</span>
	</span>
</TabsPrimitive.Trigger>
