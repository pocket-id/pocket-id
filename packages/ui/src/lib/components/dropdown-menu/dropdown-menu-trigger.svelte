<script lang="ts">
	import { DropdownMenu as DropdownMenuPrimitive } from 'bits-ui';

	let { ref = $bindable(null), ...restProps }: DropdownMenuPrimitive.TriggerProps = $props();
</script>

<!-- bits-ui opens on touch pointerup, so the tap's click would land on the phone bottom sheet that now covers the trigger and pick an item -->
<DropdownMenuPrimitive.Trigger
	bind:ref
	data-slot="dropdown-menu-trigger"
	ontouchend={(e) => e.preventDefault()}
	{...restProps}
/>
