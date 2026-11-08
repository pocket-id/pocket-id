<script lang="ts">
	import { page } from '$app/state';
	import Logo from '#lib/components/logo.svelte';
	import * as Tooltip from '@pocket-id/ui/tooltip';
	import { m } from '#lib/paraglide/messages.js';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import { commandPalette } from '#lib/stores/command-palette-store.svelte.ts';
	import type { AppVersionInformation } from '#lib/types/application-configuration.type.ts';
	import { modKey } from '#lib/utils/keyboard-util.ts';
	import { cn } from '@pocket-id/ui/utils';
	import PanelLeftCloseIcon from '~icons/solar/panel-left-close-linear';
	import PanelLeftOpenIcon from '~icons/solar/panel-left-open-linear';
	import CircleChevronUp from '~icons/solar/round-alt-arrow-up-linear';
	import Search from '~icons/solar/magnifier-linear';
	import AccountMenu from '../account-menu.svelte';
	import { activeHref, type NavGroup } from '../nav-items.ts';

	let {
		groups,
		versionInformation,
		showUpdate,
		collapsed = $bindable(),
		class: className
	}: {
		groups: NavGroup[];
		versionInformation: AppVersionInformation;
		showUpdate: boolean;
		collapsed: boolean;
		class?: string;
	} = $props();

	const current = $derived(activeHref(groups, page.url.pathname));

	const versionLabel = $derived(`Pocket ID ${versionInformation.currentVersion}`);

	// Width, height and padding glide together, so the rows slide into place in step with the sidebar edge
	const MOTION = 'duration-(--dur-slow) ease-(--ease-out)';

	// Row content keeps its expanded layout and the narrowing row clips it, so labels never re-truncate mid-glide
	const ROW_CONTENT = 'flex w-[calc(var(--sidebar-w)-1.5rem)] shrink-0 items-center';

	// Sidebar text fades out at once and fades back in once the sidebar has made room for it
	const sidebarFade = $derived(
		cn(
			'transition-[opacity,visibility] ease-(--ease-out)',
			collapsed
				? 'opacity-0 duration-(--dur-fast)'
				: 'duration-(--dur-fast) max-lg:opacity-0 lg:delay-(--dur-fast) lg:duration-(--dur-base)'
		)
	);

	// The rail's dividers and avatar fade the opposite way
	const railFade = $derived(
		cn(
			'transition-[opacity,visibility] ease-(--ease-out)',
			collapsed
				? 'delay-(--dur-fast) duration-(--dur-base)'
				: 'delay-(--dur-fast) duration-(--dur-base) lg:opacity-0 lg:delay-0 lg:duration-(--dur-fast)'
		)
	);

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

	// Collapsing, expanding and crossing a rail breakpoint resize the nav on every frame of the glide, so the fill follows its row without sliding on its own
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
		'bg-sidebar sticky top-0 hidden h-dvh w-(--rail-w) flex-col overflow-hidden px-3 pt-4 pb-3 shadow-[inset_-1px_0_0_var(--border)] transition-[width] sm:flex',
		MOTION,
		!collapsed && 'lg:w-(--sidebar-w)',
		className
	)}
>
	<div
		class={cn(
			'flex h-[42px] shrink-0 items-center pl-1.5 transition-[padding]',
			MOTION,
			!collapsed && 'lg:pl-2'
		)}
	>
		<a
			href="/"
			class="focus-visible:outline-ring flex min-w-0 items-center gap-2 rounded-sm focus-visible:outline-2 focus-visible:outline-offset-2"
			aria-label={$appConfigStore.appName}
		>
			<Logo class="size-6 shrink-0" />
			<span
				class={cn(
					'text-heading max-w-[calc(var(--sidebar-w)-4rem)] shrink-0 truncate',
					sidebarFade
				)}
			>
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
						'text-fg-muted hover:bg-fill-hover hover:text-fg focus-visible:outline-ring mt-3 flex h-9 shrink-0 overflow-hidden rounded-sm outline-offset-[-2px] transition-colors focus-visible:outline-2',
						!collapsed && 'lg:bg-fill-muted lg:hover:bg-fill-hover'
					)}
				>
					<span
						class={cn(
							ROW_CONTENT,
							'gap-2 pr-3 pl-2.5 transition-[padding]',
							MOTION,
							!collapsed && 'lg:pl-3'
						)}
					>
						<Search class="size-4 shrink-0" />
						<span class={cn('text-control flex-1 truncate text-left', sidebarFade)}
							>{m.search_or_jump_to()}</span
						>
						<span class={cn('flex gap-0.5', sidebarFade)} aria-hidden="true">
							<kbd class="keycap">{modKey.replace('+', '')}</kbd><kbd class="keycap">K</kbd>
						</span>
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
			<!-- A group's heading in the sidebar becomes a short divider in the rail, and the slot's height glides between the two -->
			<div
				class={cn(
					'relative shrink-0 overflow-hidden transition-[height]',
					MOTION,
					gi === 0 ? 'h-0' : 'h-[25px]',
					!collapsed && (gi === 0 ? 'lg:h-[38px]' : 'lg:h-[46px]')
				)}
			>
				<h2
					class={cn(
						'text-fg-muted absolute bottom-0 left-0 w-[calc(var(--sidebar-w)-1.5rem)] truncate px-4 pb-2 text-[13px] leading-[18px] font-semibold',
						sidebarFade
					)}
				>
					{group.label}
				</h2>
				{#if gi > 0}
					<span
						aria-hidden="true"
						class={cn('bg-border absolute bottom-3 left-[calc(50%-12px)] h-px w-6', railFade)}
					></span>
				{/if}
			</div>
			<ul class="flex flex-col gap-0.5">
				{#each group.items as item (item.href)}
					{@const active = item.href === current}
					{@const iconClass = cn(
						'size-[17px] shrink-0 transition-colors',
						active ? 'text-primary-control' : 'text-fg-muted group-hover:text-fg'
					)}
					<li>
						<Tooltip.Root>
							<Tooltip.Trigger>
								{#snippet child({ props })}
									<a
										{...props}
										href={item.href}
										aria-current={active ? 'page' : undefined}
										class={cn(
											'group text-fg focus-visible:outline-ring relative flex h-9 scroll-my-6 overflow-hidden rounded-sm text-sm font-medium outline-offset-[-2px] focus-visible:outline-2',
											'before:bg-fill-hover before:absolute before:inset-0 before:-z-10 before:rounded-[inherit] before:opacity-0 before:transition-opacity before:duration-(--dur-fast) hover:before:opacity-100',
											active && 'hover:before:opacity-0'
										)}
									>
										<span
											class={cn(
												ROW_CONTENT,
												'gap-3 pr-3 pl-[9.5px] transition-[padding]',
												MOTION,
												!collapsed && 'lg:pl-4'
											)}
										>
											<item.icon class={iconClass} />
											<span class={cn('grid min-w-0 flex-1', sidebarFade)}>
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

	<div class="flex shrink-0 flex-col gap-0.5 pt-2 shadow-[inset_0_1px_0_var(--border)]">
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
								'text-fg hover:bg-fill-hover focus-visible:outline-ring flex h-9 overflow-hidden rounded-sm text-sm outline-offset-[-2px] transition-[background-color,height] focus-visible:outline-2',
								MOTION,
								!collapsed && 'lg:h-12'
							)}
						>
							<span
								class={cn(
									ROW_CONTENT,
									'gap-3 pr-2 pl-[9.5px] transition-[padding]',
									MOTION,
									!collapsed && 'lg:pl-4'
								)}
							>
								<CircleChevronUp class="size-[17px] shrink-0" />
								<span class={cn('flex min-w-0 flex-col', sidebarFade)}>
									{m.update_available()}
									<span class="text-fg-muted truncate text-xs tabular-nums"
										>Pocket ID {versionInformation.newestVersion}</span
									>
								</span>
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
		{/if}
		<!-- The sidebar's version label and the rail's avatar share a slot whose height glides between the two -->
		<div
			class={cn(
				'grid h-[42px] shrink-0 overflow-hidden transition-[height]',
				MOTION,
				!collapsed && (showUpdate ? 'lg:h-0.5' : 'lg:h-6')
			)}
		>
			{#if !showUpdate}
				<p
					class={cn(
						'text-fg-muted col-start-1 row-start-1 truncate px-4 text-xs leading-5 tabular-nums',
						sidebarFade
					)}
				>
					{versionLabel}
				</p>
			{/if}
			<div class={cn('col-start-1 row-start-1 pt-1', railFade, !collapsed && 'lg:invisible')}>
				<AccountMenu variant="rail" version={versionLabel} />
			</div>
		</div>
		<!-- The toggle stays on one line with the account row, so it rides the sidebar edge instead of jumping -->
		<div class="flex items-center max-lg:hidden">
			<div class={cn('min-w-0 flex-1 overflow-hidden', sidebarFade, collapsed && 'invisible')}>
				<div class="w-[calc(var(--sidebar-w)-4rem)]">
					<AccountMenu variant="row" version={versionLabel} />
				</div>
			</div>
			<Tooltip.Root>
				<Tooltip.Trigger>
					{#snippet child({ props })}
						<button
							{...props}
							type="button"
							class="text-fg-muted hover:bg-fill-hover hover:text-fg focus-visible:outline-ring flex size-9 shrink-0 items-center justify-center rounded-sm outline-offset-[-2px] focus-visible:outline-2"
							aria-label={collapsed ? m.expand_sidebar() : m.collapse_sidebar()}
							onclick={() => (collapsed = !collapsed)}
						>
							{#if collapsed}
								<PanelLeftOpenIcon class="size-[17px]" />
							{:else}
								<PanelLeftCloseIcon class="size-[17px]" />
							{/if}
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
