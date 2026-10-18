<script lang="ts" generics="T">
	import {
		buttonVariants,
		type ButtonSize,
		type ButtonVariant
	} from '#lib/components/ui/button/index.ts';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import type { AdvancedTableAction } from '#lib/types/advanced-table.type.ts';
	import { LucideEllipsis } from '@lucide/svelte';

	let {
		item,
		actions,
		label,
		variant = 'ghost',
		size = 'icon'
	}: {
		item: T;
		actions: AdvancedTableAction<T>[];
		label: string;
		variant?: ButtonVariant;
		size?: ButtonSize;
	} = $props();

	let visibleActions = $derived(actions.filter((a) => !a.hidden));
</script>

{#if visibleActions.length > 0}
	<DropdownMenu.Root>
		<DropdownMenu.Trigger class={buttonVariants({ variant, size })}>
			<LucideEllipsis class="size-4" />
			<span class="sr-only">{label}</span>
		</DropdownMenu.Trigger>
		<DropdownMenu.Content align="end">
			{#each visibleActions as action, i (action.label)}
				{#if action.variant === 'danger' && i > 0 && visibleActions[i - 1].variant !== 'danger'}
					<DropdownMenu.Separator />
				{/if}
				<DropdownMenu.Item
					onclick={() => action.onClick(item)}
					disabled={action.disabled}
					variant={action.variant === 'danger' ? 'destructive' : 'default'}
				>
					{#if action.icon}
						{@const Icon = action.icon}
						<Icon />
					{/if}
					{action.label}
				</DropdownMenu.Item>
			{/each}
		</DropdownMenu.Content>
	</DropdownMenu.Root>
{/if}
