<script lang="ts" module>
	import { createContext } from 'svelte';

	export const [getCloseMenu, setCloseMenu] = createContext<() => void>();
</script>

<script lang="ts">
	import { DropdownMenu as DropdownMenuPrimitive } from 'bits-ui';

	let {
		open = $bindable(false),
		onOpenChange,
		...restProps
	}: DropdownMenuPrimitive.RootProps = $props();

	// bits-ui only reports its own state changes, so a close from the content reports itself
	setCloseMenu(() => {
		open = false;
		onOpenChange?.(false);
	});
</script>

<DropdownMenuPrimitive.Root bind:open {onOpenChange} {...restProps} />
