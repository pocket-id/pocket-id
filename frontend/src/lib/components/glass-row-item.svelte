<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Tooltip from '$lib/components/ui/tooltip/index.js';
	import { m } from '$lib/paraglide/messages';
	import { LucideCalendar, LucidePencil, LucideTrash, type Icon as IconType } from '@lucide/svelte';

	let {
		icon,
		onRename,
		onDelete,
		label,
		description
	}: {
		icon: typeof IconType;
		onRename: () => void;
		onDelete: () => void;
		description?: string;
		label?: string;
	} = $props();
</script>

<div class="bg-card hover:bg-muted/50 group rounded-lg p-3 transition-colors">
	<div class="flex items-center justify-between">
		<div class="flex items-start gap-3">
			<div class="bg-primary/10 text-primary mt-1 rounded-lg p-2">
				{#if icon}{@const Icon = icon}
					<Icon class="size-5" />
				{/if}
			</div>
			<div>
				<div class="flex items-center gap-2">
					<p class="font-medium">{label}</p>
				</div>
				{#if description}
					<div class="text-muted-foreground mt-1 flex items-center text-xs">
						<LucideCalendar class="mr-1 size-3" />
						{description}
					</div>
				{/if}
			</div>
		</div>

		<div class="flex items-center gap-2 opacity-0 transition-opacity group-hover:opacity-100">
			<Tooltip.Provider>
				<Tooltip.Root>
					<Tooltip.Trigger>
						<Button
							onclick={onRename}
							size="icon"
							variant="ghost"
							class="size-8"
							aria-label={m.rename()}
						>
							<LucidePencil class="size-4" />
						</Button>
					</Tooltip.Trigger>
					<Tooltip.Content>{m.rename()}</Tooltip.Content>
				</Tooltip.Root></Tooltip.Provider
			>

			<Tooltip.Provider>
				<Tooltip.Root>
					<Tooltip.Trigger>
						<Button
							onclick={onDelete}
							size="icon"
							variant="ghost"
							class="hover:bg-destructive/10 hover:text-destructive size-8"
							aria-label={m.delete()}
						>
							<LucideTrash class="size-4" />
						</Button>
					</Tooltip.Trigger>
					<Tooltip.Content>{m.delete()}</Tooltip.Content>
				</Tooltip.Root></Tooltip.Provider
			>
		</div>
	</div>
</div>
