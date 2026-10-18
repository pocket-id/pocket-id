<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import Logo from '#lib/components/logo.svelte';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import { cn } from '#lib/utils/style.ts';
	import { Check, Ellipsis } from '@lucide/svelte';
	import AccountMenu from './account-menu.svelte';
	import { activeHref, type NavGroup } from './nav-items.ts';

	let {
		groups,
		version,
		newestVersion
	}: { groups: NavGroup[]; version: string; newestVersion: string | null } = $props();

	const current = $derived(activeHref(groups, page.url.pathname));
	const [accountGroup, ...moreGroups] = $derived(groups);
	const moreActive = $derived(moreGroups.some((g) => g.items.some((i) => i.href === current)));
	let jumping = $state(false);

	function jump(href: string) {
		jumping = true;
		goto(href);
	}
</script>

<header
	data-slot="phone-top-bar"
	class="bg-sidebar sticky top-0 z-(--z-nav) flex h-[52px] items-center justify-between gap-3 px-4 [view-transition-name:phone-top-bar] sm:hidden"
>
	<a
		href="/"
		class="focus-visible:outline-ring flex min-w-0 items-center gap-2 rounded-sm focus-visible:outline-2 focus-visible:outline-offset-2"
	>
		<Logo class="size-6 shrink-0" />
		<span class="text-heading truncate">{$appConfigStore.appName}</span>
	</a>
	<AccountMenu variant="avatar" />
</header>

<nav
	aria-label={m.main()}
	class="bg-surface-raised border-border shadow-overlay fixed inset-x-3 bottom-[calc(12px+env(safe-area-inset-bottom))] z-(--z-nav) flex h-14 items-stretch rounded-lg border p-1 sm:hidden"
>
	{#each accountGroup.items as item (item.href)}
		{@render tab(item.href, item.label, item.icon, item.href === current)}
	{/each}
	{#if moreGroups.length}
		<DropdownMenu.Root
			onOpenChange={(open) => {
				if (open) jumping = false;
			}}
		>
			<DropdownMenu.Trigger
				class={cn(
					'focus-visible:outline-ring flex flex-1 flex-col items-center justify-center gap-0.5 rounded-md text-[11px] leading-[14px] focus-visible:outline-2 focus-visible:-outline-offset-2 max-[359px]:flex-auto',
					moreActive ? 'bg-fill-selected text-fg font-medium' : 'text-fg-muted'
				)}
			>
				<Ellipsis class="size-5" />
				{m.more()}
			</DropdownMenu.Trigger>
			<DropdownMenu.Content side="top" align="end" class="w-64" data-jumping={jumping || undefined}>
				{#if newestVersion}
					<DropdownMenu.Group>
						<DropdownMenu.Item>
							{#snippet child({ props })}
								<a
									{...props}
									href="https://github.com/pocket-id/pocket-id/releases/latest"
									target="_blank"
									rel="noopener noreferrer"
								>
									<CircleChevronUp />
									<span class="flex flex-col">
										{m.update_available()}
										<span class="text-fg-muted text-xs tabular-nums">Pocket ID {newestVersion}</span
										>
									</span>
								</a>
							{/snippet}
						</DropdownMenu.Item>
					</DropdownMenu.Group>
				{/if}
				{#each moreGroups as group (group.label)}
					<DropdownMenu.Group>
						<DropdownMenu.GroupHeading>{group.label}</DropdownMenu.GroupHeading>
						{#each group.items as item (item.href)}
							<DropdownMenu.Item
								onclick={() => jump(item.href)}
								aria-current={item.href === current ? 'page' : undefined}
							>
								<item.icon />
								<span class="flex-1">{item.label}</span>
								{#if item.href === current}<Check />{/if}
							</DropdownMenu.Item>
						{/each}
					</DropdownMenu.Group>
				{/each}
				{#if !newestVersion}
					<p class="text-fg-muted px-1 py-1.5 text-xs tabular-nums">{version}</p>
				{/if}
			</DropdownMenu.Content>
		</DropdownMenu.Root>
	{/if}
</nav>

{#snippet tab(
	href: string,
	label: string,
	Icon: NavGroup['items'][number]['icon'],
	active: boolean
)}
	<a
		{href}
		aria-current={active ? 'page' : undefined}
		class={cn(
			'focus-visible:outline-ring flex min-w-0 flex-1 flex-col items-center justify-center gap-0.5 rounded-md text-[11px] leading-[14px] transition-colors duration-(--dur-fast) focus-visible:outline-2 focus-visible:-outline-offset-2 max-[359px]:flex-auto',
			active ? 'bg-fill-selected text-fg font-medium' : 'text-fg-muted'
		)}
	>
		<Icon class="size-5" />
		<span class="max-w-full truncate px-1">{label}</span>
	</a>
{/snippet}
