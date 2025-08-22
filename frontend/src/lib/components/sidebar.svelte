<script lang="ts">
	import { page } from '$app/state';
	import { cn } from '$lib/utils/style';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import { m } from '$lib/paraglide/messages';
	import { LucideChevronDown, LucideExternalLink, type Icon as IconType } from '@lucide/svelte';
	import { slide } from 'svelte/transition';
	import { PersistedState } from 'runed';

	type NavItem = {
		label: string;
		href?: string;
		external?: boolean;
		icon?: typeof IconType;
		children?: NavItem[];
		id?: string;
		defaultOpen?: boolean;
	};

	let {
		items = [] as NavItem[],
		storageKey = 'sidebar-open:settings',
		isAdmin = false,
		isUpToDate = undefined,
		updateUrl = 'https://github.com/pocket-id/pocket-id/releases/latest'
	} = $props();

	const openState = new PersistedState<Record<string, boolean>>(
		storageKey,
		{},
		{
			storage: 'local',
			syncTabs: true
		}
	);

	function groupId(item: NavItem, idx: number) {
		return item.id ?? `${item.label}-${idx}`;
	}

	function isActive(href?: string) {
		if (!href) return false;
		return page.url.pathname.startsWith(href);
	}

	$effect(() => {
		const state = openState.current;

		items.forEach((item, idx) => {
			if (!item.children?.length) return;
			const id = groupId(item, idx);
			if (state[id] === undefined) {
				state[id] = item.children.some((c) => isActive(c.href)) || !!item.defaultOpen;
			}
		});
	});

	function isOpen(id: string) {
		return !!openState.current[id];
	}

	function toggle(id: string) {
		openState.current[id] = !openState.current[id];
	}

	const activeClasses =
		'text-primary bg-card rounded-md px-3 py-1.5 font-medium shadow-sm transition-all';
	const inactiveClasses =
		'hover:text-foreground hover:bg-muted/70 rounded-md px-3 py-1.5 transition-all hover:-translate-y-[2px] hover:shadow-sm';
</script>

<nav class="text-muted-foreground grid gap-2 text-sm">
	{#each items as item, i}
		{#if item.children?.length}
			{@const Icon = item.icon}
			{@const id = groupId(item, i)}
			<div class="group">
				<button
					type="button"
					class={cn(
						'hover:bg-muted/70 hover:text-foreground flex w-full items-center justify-between rounded-md px-3 py-1.5 text-left transition-all',
						!$appConfigStore.disableAnimations && 'animate-fade-in'
					)}
					style={`animation-delay: ${150 + i * 50}ms;`}
					aria-expanded={isOpen(id)}
					aria-controls={`submenu-${id}`}
					onclick={() => toggle(id)}
				>
					<span class="flex items-center gap-2">
						{#if item.icon}
							<Icon class="size-4" />
						{/if}
						{item.label}
					</span>
					<LucideChevronDown
						class={cn('size-4 transition-transform', isOpen(id) ? 'rotate-180' : '')}
					/>
				</button>

				{#if isOpen(id)}
					<ul
						id={`submenu-${id}`}
						class="border-border/50 ml-2 border-l pl-2"
						transition:slide|local={{ duration: 120 }}
					>
						{#each item.children as child, j}
							{@const Icon = child.icon}
							<li>
								<a
									href={child.href}
									target={child.external ? '_blank' : undefined}
									rel={child.external ? 'noopener noreferrer' : undefined}
									class={cn(
										isActive(child.href) ? activeClasses : inactiveClasses,
										'my-1 block',
										!$appConfigStore.disableAnimations && 'animate-fade-in'
									)}
									style={`animation-delay: ${j * 30}ms;`}
								>
									<span class="flex items-center gap-2">
										{#if child.icon}
											<Icon class="size-4" />
										{/if}
										{child.label}
									</span>
								</a>
							</li>
						{/each}
					</ul>
				{/if}
			</div>
		{:else}
			{@const Icon = item.icon}
			<a
				href={item.href}
				target={item.external ? '_blank' : undefined}
				rel={item.external ? 'noopener noreferrer' : undefined}
				class={cn(
					isActive(item.href) ? activeClasses : inactiveClasses,
					!$appConfigStore.disableAnimations && 'animate-fade-in'
				)}
				style={`animation-delay: ${150 + i * 50}ms;`}
			>
				<span class="flex items-center gap-2">
					{#if item.icon}
						<Icon class="size-4" />
					{/if}
					{item.label}
				</span>
			</a>
		{/if}
	{/each}

	{#if isAdmin && isUpToDate === false}
		<a
			href={updateUrl}
			target="_blank"
			rel="noopener noreferrer"
			class={cn(
				inactiveClasses,
				'flex items-center gap-2 text-orange-500 hover:text-orange-500/90',
				!$appConfigStore.disableAnimations && 'animate-fade-in'
			)}
			style="animation-delay: 200ms;"
		>
			{m.update_pocket_id()}
			<LucideExternalLink class="my-auto inline-block size-3" />
		</a>
	{/if}
</nav>
