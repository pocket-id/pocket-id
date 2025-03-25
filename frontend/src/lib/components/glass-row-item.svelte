<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { Tooltip, TooltipContent, TooltipTrigger } from '$lib/components/ui/tooltip';
	import { Badge } from '$lib/components/ui/badge';
	import { LucideCalendar, LucidePencil, LucideTrash, type Icon as IconType } from 'lucide-svelte';
	import type { Passkey } from '$lib/types/passkey.type';
	import { m } from '$lib/paraglide/messages';

	let {
		item,
		icon,
		onRename,
		onDelete,
		dateLabel = m.added_on(),
		primaryField = 'name',
		dateField = 'createdAt'
	}: {
		item: Passkey;
		icon: typeof IconType;
		onRename: (item: Passkey) => void;
		onDelete: (item: Passkey) => void;
		dateLabel?: string;
		primaryField?: string;
		dateField?: string;
	} = $props();

	// Function to safely access potentially nested properties by dot notation
	function getProperty(obj: any, path: string): any {
		return path.split('.').reduce((prev, curr) => (prev ? prev[curr] : null), obj);
	}

	// Use $derived to make these values reactive to item changes
	const displayName = $derived(getProperty(item, primaryField));
	const dateValue = $derived(getProperty(item, dateField));
	const formattedDate = $derived(dateValue ? new Date(dateValue).toLocaleDateString() : '');
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
					<p class="font-medium">{displayName}</p>
				</div>
				{#if formattedDate}
					<div class="text-muted-foreground mt-1 flex items-center text-xs">
						<LucideCalendar class="mr-1 h-3 w-3" />
						{dateLabel}
						{formattedDate}
					</div>
				{/if}
			</div>
		</div>

		<div class="flex items-center gap-2 opacity-0 transition-opacity group-hover:opacity-100">
			<Tooltip>
				<TooltipTrigger asChild>
					<Button
						on:click={() => onRename(item)}
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
						on:click={() => onDelete(item)}
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
