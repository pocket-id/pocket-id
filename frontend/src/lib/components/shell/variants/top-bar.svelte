<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import Logo from '#lib/components/logo.svelte';
	import * as DropdownMenu from '@pocket-id/ui/dropdown-menu';
	import * as Tooltip from '@pocket-id/ui/tooltip';
	import { m } from '#lib/paraglide/messages.js';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import { commandPalette } from '#lib/stores/command-palette-store.svelte.ts';
	import { modKey } from '#lib/utils/keyboard-util.ts';
	import { cn } from '@pocket-id/ui/utils';
	import ChevronDown from '~icons/solar/alt-arrow-down-linear';
	import Check from '~icons/solar/check-linear';
	import Search from '~icons/solar/magnifier-linear';
	import CircleChevronUp from '~icons/solar/round-alt-arrow-up-linear';
	import { tick, untrack } from 'svelte';
	import AccountMenu from '../account-menu.svelte';
	import { activeHref } from '../nav-items.ts';
	import PhoneNav from '../phone-nav.svelte';
	import type { ShellProps } from './registry.ts';

	let { groups, versionInformation, showUpdate, children }: ShellProps = $props();

	const current = $derived(activeHref(groups, page.url.pathname));
	const versionLabel = $derived(`Pocket ID ${versionInformation.currentVersion}`);

	// The header and every page share one column with the page's own gutters, so the logo, the first tab and the page title start on the same line
	const COLUMN = 'mx-auto w-full max-w-[calc(var(--col-content)+5rem)]';
	const GUTTER = 'px-8 xl:px-10';

	const TAB =
		'focus-visible:outline-ring flex h-9 shrink-0 items-center gap-2 rounded-sm px-3 text-sm font-medium whitespace-nowrap outline-offset-[-2px] transition-colors duration-(--dur-fast) focus-visible:outline-2';
	const tabColor = (active: boolean) => (active ? 'text-fg' : 'text-fg-muted hover:text-fg');

	// A row too narrow for its tabs folds each group after the first into a menu
	// It is measured instead of tied to a window width, because the tabs' width depends on the language and on which features are on
	let folded = $state(false);
	// The width the unfolded tabs needed, so the row unfolds once it offers that much again
	let unfoldedWidth = 0;

	// One shared underline slides to the current tab instead of every tab drawing its own
	let navEl = $state<HTMLElement>();
	let indicator = $state<{ left: number; width: number } | null>(null);
	let indicatorGlides = $state(false);
	let placedHref: string | undefined;
	const placeIndicator = (glide: boolean) => {
		// A folded group marks its menu button, since the page's own tab is inside the menu
		const tab = navEl?.querySelector<HTMLElement>('[data-active]');
		indicator = tab ? { left: tab.offsetLeft, width: tab.offsetWidth } : null;
		indicatorGlides = glide;
	};

	async function fit() {
		const nav = navEl;
		if (!nav) return;
		// The underline is left out of the measurement, because it may still sit where the unfolded row had its tab
		const last = nav.lastElementChild as HTMLElement;
		const tabsWidth = last.offsetLeft + last.offsetWidth;
		if (!folded && tabsWidth > nav.clientWidth) {
			unfoldedWidth = tabsWidth;
			folded = true;
		} else if (folded && nav.clientWidth >= unfoldedWidth) {
			folded = false;
		}
		await tick();
		placeIndicator(false);
	}

	// The remembered width is wrong once the tabs themselves change, so the row is measured unfolded again
	async function refit() {
		folded = false;
		await tick();
		fit();
	}

	// Only a change of page slides the underline
	$effect(() => {
		const href = current;
		if (!navEl) return;
		placeIndicator(placedHref !== undefined && href !== placedHref);
		placedHref = href;
	});

	// Tabs join or leave when a feature is switched
	const tabHrefs = $derived(groups.flatMap((g) => g.items.map((i) => i.href)).join());
	$effect(() => {
		void tabHrefs;
		untrack(refit);
	});

	// The row's width follows the window
	// The tabs' own width changes once the web font has loaded and when another text size is saved in Appearance, which a tab reports while the row keeps its width
	$effect(() => {
		if (!navEl) return;
		const rowObserver = new ResizeObserver(fit);
		rowObserver.observe(navEl);
		const tabObserver = new ResizeObserver(refit);
		tabObserver.observe(navEl.querySelector('a')!);
		return () => {
			rowObserver.disconnect();
			tabObserver.disconnect();
		};
	});

	let jumping = $state(false);
</script>

<div class="top-bar bg-surface flex min-h-dvh flex-col">
	<!-- The brand row scrolls away with the page and only the tabs stay, and both wear the sidebar color so they read as one band -->
	<div
		data-slot="app-top-bar"
		class="bg-sidebar hidden [view-transition-name:top-bar-brand] sm:block"
	>
		<div class="flex h-16 items-center gap-4 {COLUMN} {GUTTER}">
			<a
				href="/"
				class="focus-visible:outline-ring flex min-w-0 items-center gap-2.5 rounded-sm focus-visible:outline-2 focus-visible:outline-offset-2"
				aria-label={$appConfigStore.appName}
			>
				<Logo class="size-6 shrink-0" />
				<span class="text-title truncate">{$appConfigStore.appName}</span>
			</a>
			<div class="flex flex-1 items-center justify-end gap-2">
				{#if showUpdate}
					<a
						href="https://github.com/pocket-id/pocket-id/releases/latest"
						target="_blank"
						rel="noopener noreferrer"
						title="Pocket ID {versionInformation.newestVersion}"
						class="text-fg hover:bg-fill-hover focus-visible:outline-ring flex h-9 shrink-0 items-center gap-2 rounded-sm px-3 text-sm font-medium transition-colors duration-(--dur-fast) focus-visible:outline-2"
					>
						<CircleChevronUp class="size-[17px]" />
						{m.update_available()}
					</a>
				{/if}
				<Tooltip.Root>
					<Tooltip.Trigger>
						{#snippet child({ props })}
							<button
								{...props}
								type="button"
								aria-label={m.search_or_jump_to()}
								aria-haspopup="dialog"
								onclick={() => (commandPalette.open = true)}
								class="text-fg-muted hover:bg-fill-hover hover:text-fg focus-visible:outline-ring flex size-9 shrink-0 items-center justify-center rounded-sm outline-offset-[-2px] transition-colors duration-(--dur-fast) focus-visible:outline-2"
							>
								<Search class="size-4" />
							</button>
						{/snippet}
					</Tooltip.Trigger>
					<Tooltip.Content side="bottom">
						{m.search_or_jump_to()}<span class="text-inverse-muted">{modKey}K</span>
					</Tooltip.Content>
				</Tooltip.Root>
				<AccountMenu variant="avatar" version={versionLabel} />
			</div>
		</div>
	</div>
	<div
		data-slot="app-top-bar"
		class="bg-sidebar sticky top-0 z-(--z-nav) hidden shadow-[inset_0_-1px_0_var(--border)] [view-transition-name:top-bar-tabs] sm:block"
	>
		<div class="{COLUMN} {GUTTER}">
			<!-- The row is pulled out by the tabs' own padding, so the first tab's content sits on the column's edge -->
			<nav
				bind:this={navEl}
				aria-label={m.main()}
				class="relative isolate -mx-3 flex h-12 items-center gap-0.5"
			>
				{#if indicator}
					<!-- Transparent side borders inset the line by the tabs' padding, so it spans the tab's content only -->
					<span
						aria-hidden="true"
						class={cn(
							'bg-primary-control absolute bottom-0 left-0 -z-10 h-0.5 border-x-12 border-transparent bg-clip-padding',
							indicatorGlides && 'transition-[transform,width] duration-(--dur-base) ease-in-out'
						)}
						style="width: {indicator.width}px; transform: translateX({indicator.left}px)"
					></span>
				{/if}
				{#each groups as group, gi (group.label)}
					{#if gi > 0}
						<span aria-hidden="true" class="bg-border mx-2 h-5 w-px shrink-0"></span>
					{/if}
					{#if gi > 0 && folded}
						{@const groupActive = group.items.some((i) => i.href === current)}
						<DropdownMenu.Root
							onOpenChange={(open) => {
								if (open) jumping = false;
							}}
						>
							<DropdownMenu.Trigger
								data-active={groupActive || undefined}
								class={cn(TAB, tabColor(groupActive), 'data-[state=open]:text-fg')}
							>
								{group.label}
								<ChevronDown class="size-3.5" />
							</DropdownMenu.Trigger>
							<DropdownMenu.Content align="start" class="w-60" data-jumping={jumping || undefined}>
								{#each group.items as item (item.href)}
									<DropdownMenu.Item
										onclick={() => {
											jumping = true;
											goto(item.href);
										}}
										aria-current={item.href === current ? 'page' : undefined}
									>
										<item.icon />
										<span class="flex-1">{item.label}</span>
										{#if item.href === current}<Check class="text-primary-control" />{/if}
									</DropdownMenu.Item>
								{/each}
							</DropdownMenu.Content>
						</DropdownMenu.Root>
					{:else}
						{#each group.items as item (item.href)}
							{@const active = item.href === current}
							<a
								href={item.href}
								aria-current={active ? 'page' : undefined}
								data-active={active || undefined}
								class={cn(TAB, tabColor(active))}
							>
								{item.label}
							</a>
						{/each}
					{/if}
				{/each}
			</nav>
		</div>
	</div>
	<PhoneNav
		{groups}
		version={versionLabel}
		newestVersion={showUpdate ? versionInformation.newestVersion : null}
	/>
	<!-- List pages run gutter to gutter beside a sidebar, here they keep to the column like every other page -->
	<div class="flex flex-1 flex-col {COLUMN}">
		{@render children()}
	</div>
</div>

<style>
	/* The overscroll bounce at the top reveals the body, which continues the band instead of flashing the canvas above it */
	:global(body:has(.top-bar)) {
		background: var(--sidebar);
	}

	/* Anchors and a page's own sticky panels stop below the tabs instead of sliding under them */
	@media (width >= 40rem) {
		:global(html:has(.top-bar)) {
			--clear-top: calc(var(--spacing) * 12);
			scroll-padding-top: var(--clear-top);
		}
	}
</style>
