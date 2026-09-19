<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Item from '$lib/components/ui/item/index.js';
	import * as Tooltip from '$lib/components/ui/tooltip/index.js';
	import { m } from '$lib/paraglide/messages';
	import { LucideCalendar, LucidePencil, LucideTrash, type LucideIcon } from '@lucide/svelte';

	let {
		icon,
		providerIconUrl,
		onRename,
		onDelete,
		showRenameAction = true,
		label,
		description
	}: {
		icon: LucideIcon;
		providerIconUrl?: string;
		onRename?: () => void;
		onDelete: () => void;
		showRenameAction?: boolean;
		description?: string;
		label?: string;
	} = $props();

	// Falls back to the generic icon when the authenticator icon cannot be loaded
	let iconFailed = $state(false);

	// The row is reused across passkeys and themes, so a previous failure must not hide an icon that is now a different URL
	$effect(() => {
		void providerIconUrl;
		iconFailed = false;
	});

	const showProviderIcon = $derived(!!providerIconUrl && !iconFailed);
</script>

<Item.Root variant="transparent" class="hover:bg-muted transition-colors py-3 px-0 sm:px-4">
	<Item.Media class="bg-muted text-muted-foreground size-11 rounded-xl">
		{#if showProviderIcon}
			<img
				src={providerIconUrl}
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
