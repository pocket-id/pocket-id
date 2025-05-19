<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { Tooltip, TooltipContent, TooltipTrigger } from '$lib/components/ui/tooltip';
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
					<Icon class="h-5 w-5" />
				{/if}
			</div>
			<div>
				<div class="flex items-center gap-2">
					<p class="font-medium">{label}</p>
				</div>
				{#if description}
					<div class="text-muted-foreground mt-1 flex items-center text-xs">
						<LucideCalendar class="mr-1 h-3 w-3" />
						{description}
					</div>
				{/if}
			</div>
		</div>

		<div class="flex items-center gap-2 opacity-0 transition-opacity group-hover:opacity-100">
			<Tooltip>
				<TooltipTrigger asChild>
					<Button
						onclick={onRename}
						size="icon"
						variant="ghost"
						class="h-8 w-8"
						aria-label={m.rename()}
					>
						<LucidePencil class="h-4 w-4" />
					</Button>
				</TooltipTrigger>
				<TooltipContent>{m.rename()}</TooltipContent>
			</Tooltip>

			<Tooltip>
				<TooltipTrigger asChild>
					<Button
						onclick={onDelete}
						size="icon"
						variant="ghost"
						class="hover:bg-destructive/10 hover:text-destructive h-8 w-8"
						aria-label={m.delete()}
					>
						<LucideTrash class="h-4 w-4" />
					</Button>
				</TooltipTrigger>
				<TooltipContent>{m.delete()}</TooltipContent>
			</Tooltip>
		</div>
	</div>
</div>
