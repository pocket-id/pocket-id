<script lang="ts">
	import { page } from '$app/state';
	import { m } from '$lib/paraglide/messages';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import { cn } from '$lib/utils/style';
	import { LucideChevronDown, LucideExternalLink } from '@lucide/svelte';
	import { PersistedState } from 'runed';
	import { slide } from 'svelte/transition';

	type NavItem = {
		href?: string;
		label: string;
		children?: NavItem[];
	};

	let {
		items = [] as NavItem[],
		storageKey = 'sidebar-open:settings',
		isAdmin = false,
		isUpToDate = undefined
	} = $props();

	const openState = new PersistedState<Record<string, boolean>>(storageKey, {});

	function groupId(item: NavItem, idx: number) {
		return `${item.label}-${idx}`;
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
				state[id] = item.children.some((c) => isActive(c.href));
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

	const ROW_STAGGER = 50;

	// Derive the offset (row index) for each top-level item,
	// counting expanded children of previous items.
	const layout = $derived(() => {
		const offsets: number[] = [];
		let total = 0;
		items.forEach((it, idx) => {
			offsets[idx] = total; // row index for this top-level item
			total += 1; // this item itself
			const id = groupId(it, idx);
			if (it.children?.length && openState.current[id]) {
				total += it.children.length; // rows for visible children
			}
		});
		return { offsets, total };
	});

	const delayTop = (i: number) => `${layout().offsets[i] * ROW_STAGGER}ms`;
	const delayChild = (i: number, j: number) => `${(layout().offsets[i] + 1 + j) * ROW_STAGGER}ms`;
	const delayUpdateLink = () => `${layout().total * ROW_STAGGER}ms`;
</script>

<nav class="text-muted-foreground grid gap-2 text-sm">
	{#each items as item, i}
		{#if item.children?.length}
			{@const id = groupId(item, i)}
			<div class="group">
				<button
					type="button"
					class={cn(
						'hover:bg-muted/70 hover:text-foreground flex w-full items-center justify-between rounded-md px-3 py-1.5 text-left transition-all',
						!$appConfigStore.disableAnimations && 'animate-fade-in'
					)}
					style={`animation-delay: ${delayTop(i)};`}
					aria-expanded={isOpen(id)}
					aria-controls={`submenu-${id}`}
					onclick={() => toggle(id)}
				>
					{item.label}

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
							<li>
								<a
									href={child.href}
									class={cn(
										isActive(child.href) ? activeClasses : inactiveClasses,
										'my-1 block',
										!$appConfigStore.disableAnimations && 'animate-fade-in'
									)}
									style={`animation-delay: ${delayChild(i, j)};`}
								>
									{child.label}
								</a>
							</li>
						{/each}
					</ul>
				{/if}
			</div>
		{:else}
			<a
				href={item.href}
				class={cn(
					isActive(item.href) ? activeClasses : inactiveClasses,
					!$appConfigStore.disableAnimations && 'animate-fade-in'
				)}
				style={`animation-delay: ${delayTop(i)};`}
			>
				{item.label}
			</a>
		{/if}
	{/each}

	{#if isAdmin && isUpToDate === false}
		<a
			href="https://github.com/pocket-id/pocket-id/releases/latest"
			target="_blank"
			rel="noopener noreferrer"
			class={cn(
				inactiveClasses,
				'flex items-center gap-2 text-orange-500 hover:text-orange-500/90',
				!$appConfigStore.disableAnimations && 'animate-fade-in'
			)}
			style={`animation-delay: ${delayUpdateLink()};`}
		>
			{m.update_pocket_id()}
			<LucideExternalLink class="my-auto inline-block size-3" />
		</a>
	{/if}
</nav>
