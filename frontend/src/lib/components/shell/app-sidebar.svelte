<script lang="ts">
	import { page } from '$app/state';
	import Logo from '#lib/components/logo.svelte';
	import * as Tooltip from '#lib/components/ui/tooltip/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import { commandPalette } from '#lib/stores/command-palette-store.svelte.ts';
	import type { AppVersionInformation } from '#lib/types/application-configuration.type.ts';
	import { modKey } from '#lib/utils/keyboard-util.ts';
	import { cn } from '#lib/utils/style.ts';
	import { CircleChevronUp, PanelLeftClose, PanelLeftOpen, Search } from '@lucide/svelte';
	import AccountMenu from './account-menu.svelte';
	import { activeHref, type NavGroup } from './nav-items.ts';

	let {
		groups,
		versionInformation,
		showUpdate,
		collapsed = $bindable()
	}: {
		groups: NavGroup[];
		versionInformation: AppVersionInformation;
		showUpdate: boolean;
		collapsed: boolean;
	} = $props();

	const current = $derived(activeHref(groups, page.url.pathname));
	const versionLabel = $derived(`Pocket ID ${versionInformation.currentVersion}`);

	// One shared selected fill slides to the current row instead of every row painting its own
	let navEl = $state<HTMLElement>();
	let indicator = $state<{ top: number; left: number; width: number } | null>(null);
	let indicatorGlides = $state(false);
	let placedHref: string | undefined;
	const placeIndicator = (nav: HTMLElement, href: string | undefined, glide: boolean) => {
		const row = nav.querySelector<HTMLElement>(`a[href="${href}"]`);
		indicator = row ? { top: row.offsetTop, left: row.offsetLeft, width: row.offsetWidth } : null;
		indicatorGlides = glide;
		placedHref = href;
		return row;
	};

	// APIs and API keys rows join or leave without a page change and move the rows below them
	const rowHrefs = $derived(groups.flatMap((g) => g.items.map((i) => i.href)).join());

	// Only a change of page slides the fill
	// On short windows the nav scrolls, and the rows' scroll margin brings the current one clear of the edge fade
	$effect(() => {
		void rowHrefs;
		const nav = navEl;
		const href = current;
		if (!nav) return;
		const row = placeIndicator(nav, href, placedHref !== undefined && href !== placedHref);
		row?.scrollIntoView({ block: 'nearest' });
	});

	// Collapsing, expanding and crossing a rail breakpoint resize the nav and move its rows, so the fill jumps along with its row
	$effect(() => {
		const nav = navEl;
		if (!nav) return;
		const observer = new ResizeObserver(() => placeIndicator(nav, placedHref, false));
		observer.observe(nav);
		return () => observer.disconnect();
	});
</script>

<aside
	data-slot="app-sidebar"
	class={cn(
		'bg-sidebar sticky top-0 hidden h-dvh flex-col px-3 pt-4 pb-3 sm:flex',
		collapsed
			? 'w-(--rail-w) items-center px-2'
			: 'w-(--rail-w) items-center px-2 lg:w-(--sidebar-w) lg:items-stretch lg:px-3'
	)}
>
	<div class={cn('flex h-[42px] shrink-0 items-center gap-2', !collapsed && 'lg:pl-2')}>
		<a
			href="/"
			class="focus-visible:outline-ring flex min-w-0 items-center gap-2 rounded-sm focus-visible:outline-2 focus-visible:outline-offset-2"
			aria-label={$appConfigStore.appName}
		>
			<Logo class="size-6 shrink-0" />
			<span class={cn('text-heading truncate', collapsed ? 'hidden' : 'hidden lg:inline')}>
				{$appConfigStore.appName}
			</span>
		</a>
	</div>

	<!-- A field-like trigger in the sidebar, a search icon in the rail -->
	<Tooltip.Root>
		<Tooltip.Trigger>
			{#snippet child({ props })}
				<button
					{...props}
					type="button"
					aria-label={m.search_or_jump_to()}
					aria-haspopup="dialog"
					onclick={() => (commandPalette.open = true)}
					class={cn(
						'text-fg-muted focus-visible:outline-ring mt-3 flex h-9 shrink-0 items-center gap-2 rounded-sm outline-offset-[-2px] transition-colors focus-visible:outline-2',
						collapsed
							? 'hover:bg-fill-hover hover:text-fg w-9 justify-center'
							: 'hover:bg-fill-hover hover:text-fg lg:bg-fill-muted lg:hover:bg-fill-hover w-9 justify-center lg:w-full lg:justify-start lg:px-3'
					)}
				>
					<Search class="size-4 shrink-0" />
					<span
						class={cn(
							'text-control flex-1 truncate text-left',
							collapsed ? 'hidden' : 'hidden lg:inline'
						)}>{m.search_or_jump_to()}</span
					>
					<span class={cn('gap-0.5', collapsed ? 'hidden' : 'hidden lg:flex')} aria-hidden="true">
						<kbd class="keycap">{modKey.replace('+', '')}</kbd><kbd class="keycap">K</kbd>
					</span>
				</button>
			{/snippet}
		</Tooltip.Trigger>
		<Tooltip.Content side="right" class={cn(!collapsed && 'lg:hidden')}>
			{m.search_or_jump_to()}<span class="text-inverse-muted">{modKey}K</span>
		</Tooltip.Content>
	</Tooltip.Root>

	<nav
		bind:this={navEl}
		aria-label={m.main()}
		class="scroll-fade-y relative isolate -mx-3 mt-1 flex min-h-0 w-[calc(100%+1.5rem)] flex-1 flex-col overflow-x-hidden overflow-y-auto px-3 pb-4"
	>
		{#if indicator}
			<span
				aria-hidden="true"
				class={cn(
					'bg-fill-selected absolute -z-10 h-9 rounded-sm',
					indicatorGlides && 'transition-transform duration-(--dur-base) ease-in-out'
				)}
				style="left: {indicator.left}px; width: {indicator.width}px; transform: translateY({indicator.top}px); top: 0"
			></span>
		{/if}
		{#each groups as group, gi (group.label)}
			<h2
				class={cn(
					'text-fg-muted px-4 pb-2 text-[13px] leading-[18px] font-semibold',
					gi === 0 ? 'pt-3' : 'pt-5',
					collapsed ? 'sr-only' : 'max-lg:sr-only'
				)}
			>
				{group.label}
			</h2>
			{#if gi > 0}
				<span
					aria-hidden="true"
					class={cn('bg-border mx-auto my-3 h-px w-6', !collapsed && 'lg:hidden')}
				></span>
			{/if}
			<ul class={cn('flex flex-col gap-0.5', collapsed ? 'items-center' : 'max-lg:items-center')}>
				{#each group.items as item (item.href)}
					{@const active = item.href === current}
					<li>
						<Tooltip.Root>
							<Tooltip.Trigger>
								{#snippet child({ props })}
									<a
										{...props}
										href={item.href}
										aria-current={active ? 'page' : undefined}
										class={cn(
											'group text-fg focus-visible:outline-ring relative flex h-9 items-center gap-3 rounded-sm text-sm font-medium outline-offset-[-2px] focus-visible:outline-2',
											'before:bg-fill-hover before:absolute before:inset-0 before:-z-10 before:rounded-[inherit] before:opacity-0 before:transition-opacity before:duration-(--dur-fast) hover:before:opacity-100',
											active && 'hover:before:opacity-0',
											collapsed
												? 'w-9 justify-center'
												: 'w-9 justify-center lg:w-full lg:justify-start lg:pr-3 lg:pl-4'
										)}
									>
										<item.icon
											class={cn(
												'size-[17px] shrink-0 transition-colors',
												active ? 'text-fg' : 'text-fg-muted group-hover:text-fg'
											)}
										/>
										<span
											class={cn(
												'min-w-0 flex-1',
												collapsed ? 'sr-only' : 'max-lg:sr-only lg:inline-grid'
											)}
										>
											<span
												class={cn('col-start-1 row-start-1 truncate', active && 'opacity-0')}
												aria-hidden={active}>{item.label}</span
											>
											<span
												aria-hidden={!active}
												class={cn(
													'col-start-1 row-start-1 truncate font-semibold',
													!active && 'invisible'
												)}>{item.label}</span
											>
										</span>
									</a>
								{/snippet}
							</Tooltip.Trigger>
							<Tooltip.Content side="right" class={cn(!collapsed && 'lg:hidden')}
								>{item.label}</Tooltip.Content
							>
						</Tooltip.Root>
					</li>
				{/each}
			</ul>
		{/each}
	</nav>

	<div class="flex w-full shrink-0 flex-col gap-0.5 pt-2 shadow-[inset_0_1px_0_var(--border)]">
		{#if showUpdate}
			<Tooltip.Root>
				<Tooltip.Trigger>
					{#snippet child({ props })}
						<a
							{...props}
							href="https://github.com/pocket-id/pocket-id/releases/latest"
							target="_blank"
							rel="noopener noreferrer"
							class={cn(
								'text-fg hover:bg-fill-hover focus-visible:outline-ring flex min-h-11 items-center gap-3 rounded-sm py-1.5 text-sm outline-offset-[-2px] transition-colors focus-visible:outline-2',
								collapsed
									? 'size-9 min-h-9 justify-center self-center'
									: 'max-lg:size-9 max-lg:min-h-9 max-lg:justify-center max-lg:self-center lg:pr-2 lg:pl-4'
							)}
						>
							<CircleChevronUp class="size-[17px] shrink-0" />
							<span class={cn('flex min-w-0 flex-col', collapsed ? 'sr-only' : 'max-lg:sr-only')}>
								{m.update_available()}
								<span class="text-fg-muted truncate text-xs tabular-nums"
									>Pocket ID {versionInformation.newestVersion}</span
								>
							</span>
						</a>
					{/snippet}
				</Tooltip.Trigger>
				<Tooltip.Content side="right" class={cn(!collapsed && 'lg:hidden')}>
					{m.update_available()}<span class="text-inverse-muted tabular-nums"
						>· Pocket ID {versionInformation.newestVersion}</span
					>
				</Tooltip.Content>
			</Tooltip.Root>
		{:else}
			<p
				class={cn(
					'text-fg-muted px-4 text-xs leading-5 tabular-nums',
					collapsed ? 'hidden' : 'max-lg:hidden'
				)}
			>
				{versionLabel}
			</p>
		{/if}
		<div class={cn('mt-1 flex items-center gap-1', collapsed ? 'flex-col' : 'max-lg:flex-col')}>
			<div class={cn('min-w-0 flex-1', collapsed ? 'hidden' : 'max-lg:hidden')}>
				<AccountMenu variant="row" version={versionLabel} />
			</div>
			<div class={cn(collapsed ? 'block' : 'lg:hidden')}>
				<AccountMenu variant="rail" version={versionLabel} />
			</div>
			<Tooltip.Root>
				<Tooltip.Trigger>
					{#snippet child({ props })}
						<button
							{...props}
							type="button"
							class="text-fg-muted hover:bg-fill-hover hover:text-fg focus-visible:outline-ring hidden size-9 shrink-0 items-center justify-center rounded-sm outline-offset-[-2px] focus-visible:outline-2 lg:flex"
							aria-label={collapsed ? m.expand_sidebar() : m.collapse_sidebar()}
							onclick={() => (collapsed = !collapsed)}
						>
							{#if collapsed}<PanelLeftOpen class="size-[17px]" />{:else}<PanelLeftClose
									class="size-[17px]"
								/>{/if}
						</button>
					{/snippet}
				</Tooltip.Trigger>
				<Tooltip.Content side="right">
					{collapsed ? m.expand_sidebar() : m.collapse_sidebar()}
				</Tooltip.Content>
			</Tooltip.Root>
		</div>
	</div>
</aside>
