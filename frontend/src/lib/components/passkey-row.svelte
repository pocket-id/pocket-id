<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Item from '$lib/components/ui/item/index.js';
	import * as Tooltip from '$lib/components/ui/tooltip/index.js';
	import { m } from '$lib/paraglide/messages';
	import {
		LucideCalendar,
		LucideImage,
		LucideImageOff,
		LucidePencil,
		LucideTrash,
		type Icon as IconType
	} from '@lucide/svelte';

	let {
		icon,
		providerIcon,
		onRename,
		onDelete,
		showRenameAction = true,
		label,
		description
	}: {
		icon: typeof IconType;
		providerIcon?: { url?: string; onToggleIcon?: () => void };
		onRename?: () => void;
		onDelete: () => void;
		showRenameAction?: boolean;
		description?: string;
		label?: string;
	} = $props();

	let iconFailed = $state(false);

	$effect(() => {
		void providerIcon?.url;
		iconFailed = false;
	});

	const showProviderIcon = $derived(!!providerIcon?.url && !iconFailed);
</script>

<Item.Root variant="transparent" class="hover:bg-muted transition-colors py-3 px-0 sm:px-4">
	<Item.Media class="bg-muted text-muted-foreground size-11 rounded-xl">
		{#if showProviderIcon}
			<img
				src={providerIcon?.url}
				alt=""
				class="size-7 object-contain"
				onerror={() => (iconFailed = true)}
			/>
		{:else if icon}{@const Icon = icon}
			<Icon class="size-6" />
		{/if}
	</Item.Media>
	<Item.Content class="gap-0.5">
		<Item.Title>{label}</Item.Title>
		{#if description}
			<Item.Description class="flex items-center">
				<LucideCalendar class="mr-1 size-3" />
				{description}
			</Item.Description>
		{/if}
	</Item.Content>
	<Item.Actions>
		{#if providerIcon?.onToggleIcon}
			{@const toggleLabel = providerIcon.url ? m.clear_icon() : m.restore_icon()}
			<Tooltip.Provider>
				<Tooltip.Root>
					<Tooltip.Trigger>
						<Button
							onclick={providerIcon.onToggleIcon}
							size="icon"
							variant="ghost"
							class="size-8"
							aria-label={toggleLabel}
						>
							{#if providerIcon.url}
								<LucideImageOff class="size-4" />
							{:else}
								<LucideImage class="size-4" />
							{/if}
						</Button>
					</Tooltip.Trigger>
					<Tooltip.Content>{toggleLabel}</Tooltip.Content>
				</Tooltip.Root>
			</Tooltip.Provider>
		{/if}

		{#if showRenameAction && onRename}
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
				</Tooltip.Root>
			</Tooltip.Provider>
		{/if}

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
			</Tooltip.Root>
		</Tooltip.Provider>
	</Item.Actions>
</Item.Root>
