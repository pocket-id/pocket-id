<script lang="ts">
	import FormattedMessage from '#lib/components/formatted-message.svelte';
	import { monogram } from '#lib/components/oidc-client-avatar.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Command from '#lib/components/ui/command/index.ts';
	import * as Popover from '#lib/components/ui/popover/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import OidcService from '#lib/services/oidc-service.ts';
	import type { AppConfig } from '#lib/types/application-configuration.type.ts';
	import type { OidcClientLogoPreset } from '#lib/types/oidc.type.ts';
	import { debounced } from '#lib/utils/debounce-util.ts';
	import { getAxiosErrorMessage } from '#lib/utils/error-util.ts';
	import { cn } from '#lib/utils/style.ts';
	import { LucideImageOff, LucideMoon, LucideSun, LucideUpload } from '@lucide/svelte';
	import { Command as CommandPrimitive } from 'bits-ui';
	import { SvelteSet } from 'svelte/reactivity';

	let {
		clientName,
		logoDataURL,
		darkLogoDataURL,
		iconLibrary,
		onLogoChange,
		onPresetSelect,
		onReset
	}: {
		clientName: string;
		logoDataURL: string | null;
		darkLogoDataURL: string | null;
		iconLibrary: AppConfig['iconLibrary'];
		onLogoChange: (input: File | string, light: boolean) => void;
		onPresetSelect: (preset: OidcClientLogoPreset) => void;
		onReset: () => void;
	} = $props();

	const oidcService = new OidcService();

	const popoverOffset = 4;
	const viewportMargin = 16;
	const lightBackground = 'bg-well-light text-well-light-fg';
	const darkBackground = 'bg-well-dark text-well-dark-fg';

	let open = $state(false);
	let alignOffset = $state(0);
	let triggerRef = $state<HTMLElement | null>(null);
	let contentRef = $state<HTMLElement | null>(null);
	let lightFileInput = $state<HTMLInputElement | null>(null);
	let darkFileInput = $state<HTMLInputElement | null>(null);
	let search = $state('');
	let presets = $state<OidcClientLogoPreset[]>([]);
	// The search the shown presets answer, so an empty result keeps its message while the next search loads instead of collapsing the popover
	let presetsQuery = $state<string | null>(null);
	let errorMessage = $state<string | null>(null);
	let latestRequest = 0;
	// An image that fails to load shows a fallback instead of the browser's broken-image icon
	const brokenLogos = new SvelteSet<string>();

	const presetsEnabled = $derived(iconLibrary !== 'disabled');
	const imageUrl = $derived(parseImageUrl(search));
	const showPresets = $derived(presetsEnabled && !imageUrl);
	const showIconGrid = $derived(showPresets && presets.length > 0);
	const effectiveLightLogoURL = $derived(logoDataURL ?? darkLogoDataURL);
	const effectiveDarkLogoURL = $derived(darkLogoDataURL ?? logoDataURL);
	const hasLogo = $derived(!!(logoDataURL || darkLogoDataURL));

	// Input that parses as an absolute HTTP(S) URL is used as the image itself instead of as a search term
	function parseImageUrl(value: string): string | null {
		const trimmed = value.trim();
		try {
			const url = new URL(trimmed);
			return url.protocol === 'http:' || url.protocol === 'https:' ? trimmed : null;
		} catch {
			return null;
		}
	}

	async function loadPresets(query: string) {
		const request = ++latestRequest;

		try {
			const result = await oidcService.searchLogoPresets(query);
			// Responses can arrive out of order, so only the newest search may update the results
			if (request === latestRequest) {
				presets = result;
				presetsQuery = query;
				errorMessage = null;
			}
		} catch (e) {
			if (request === latestRequest) {
				presets = [];
				presetsQuery = query;
				errorMessage = getAxiosErrorMessage(e);
			}
		}
	}

	// The popover always opens below the logo without collision handling, so it is fitted into the viewport by hand whenever its size or the window changes
	$effect(() => {
		if (!open || !triggerRef || !contentRef) return;

		const trigger = triggerRef;
		const content = contentRef;
		const fit = () => fitPopoverIntoViewport(trigger, content);
		const observer = new ResizeObserver(fit);
		observer.observe(content);
		window.addEventListener('resize', fit);

		return () => {
			observer.disconnect();
			window.removeEventListener('resize', fit);
		};
	});

	// Measured from the logo because the popover itself is still animating in
	function fitPopoverIntoViewport(trigger: HTMLElement, content: HTMLElement) {
		const triggerRect = trigger.getBoundingClientRect();

		// Shift the popover left when it would stick out on the right, which happens on narrow screens
		const overflowRight =
			triggerRect.left + content.offsetWidth + viewportMargin - window.innerWidth;
		alignOffset = -Math.max(0, overflowRight);

		// Scroll the page just far enough to show all of the popover
		// The scroll is instant because a smooth one gets cancelled when the results arrive and the first icon is scrolled into view
		const overflowBottom =
			triggerRect.bottom +
			popoverOffset +
			content.offsetHeight +
			viewportMargin -
			window.innerHeight;
		if (overflowBottom > 0) {
			window.scrollBy({ top: overflowBottom, behavior: 'instant' });
		}
	}

	// A search that was queued while typing is skipped once the input has changed since, or if it is a URL
	const onSearch = debounced((query: string) => {
		if (!presetsEnabled || query !== search || parseImageUrl(query)) return;
		return loadPresets(query);
	}, 250);

	function resetSearch() {
		search = '';
		if (presetsEnabled) {
			loadPresets(search);
		}
	}

	function onOpenChange(isOpen: boolean) {
		if (!isOpen) return;

		// Results of the previous search would otherwise show until the new one finishes
		presets = [];
		presetsQuery = null;
		errorMessage = null;
		resetSearch();
	}

	function selectPreset(preset: OidcClientLogoPreset) {
		open = false;
		onPresetSelect(preset);
	}

	// The picker stays open after a custom image is added, so the current logo tile shows both variants and the other one can be added too
	function selectUrl(url: string, light: boolean) {
		onLogoChange(url, light);
		resetSearch();
	}

	// Pressing enter on a pasted URL uses it as the light logo, which dark mode falls back to
	function onInputKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && imageUrl) {
			e.preventDefault();
			selectUrl(imageUrl, true);
		}
	}

	function onFileChange(e: Event, light: boolean) {
		const input = e.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		// Clearing the input lets the same file be picked again after it was removed
		input.value = '';
		if (!file) return;

		onLogoChange(file, light);
	}
</script>

<!-- The file inputs live outside the popover so a picked file is still handled after it closes -->
<input
	id="oidc-client-logo-light"
	bind:this={lightFileInput}
	type="file"
	accept="image/*"
	class="hidden"
	onchange={(e) => onFileChange(e, true)}
/>
<input
	id="oidc-client-logo-dark"
	bind:this={darkFileInput}
	type="file"
	accept="image/*"
	class="hidden"
	onchange={(e) => onFileChange(e, false)}
/>

{#snippet tileHalf(src: string | null, light: boolean)}
	<span
		class={cn('flex items-center justify-center p-2', light ? lightBackground : darkBackground)}
	>
		{#if src}
			<img {src} alt="" class="size-7 object-contain" loading="lazy" />
		{:else if light}
			<LucideSun class="size-4" />
		{:else}
			<LucideMoon class="size-4" />
		{/if}
	</span>
{/snippet}

{#snippet uploadHalf(light: boolean)}
	<button
		type="button"
		class={cn(
			'text-micro flex flex-col items-center justify-center gap-1 transition-opacity hover:opacity-80',
			light ? lightBackground : darkBackground
		)}
		aria-label={light ? m.upload_light_logo() : m.upload_dark_logo()}
		onclick={() => (light ? lightFileInput : darkFileInput)?.click()}
	>
		<LucideUpload class="size-4" />
		{light ? m.light() : m.dark()}
	</button>
{/snippet}

{#snippet urlHalf(url: string, light: boolean)}
	<button
		type="button"
		class={cn(
			'text-micro flex flex-col items-center justify-center gap-1 transition-opacity hover:opacity-80',
			light ? lightBackground : darkBackground
		)}
		aria-label={light ? m.use_as_light_logo() : m.use_as_dark_logo()}
		title={light ? m.use_as_light_logo() : m.use_as_dark_logo()}
		onclick={() => selectUrl(url, light)}
	>
		{#if brokenLogos.has(url)}
			<LucideImageOff class="size-4" />
		{:else}
			<img src={url} alt="" class="size-6 object-contain" onerror={() => brokenLogos.add(url)} />
		{/if}
		{light ? m.light() : m.dark()}
	</button>
{/snippet}

<!-- Each swatch sits on its theme's own background, and an empty dark slot borrows the light logo like the client tile does -->
{#snippet well(src: string | null, light: boolean)}
	<div class="flex w-[88px] flex-col items-center gap-1.5">
		<span
			class={cn(
				'grid size-[88px] place-items-center rounded-md shadow-[inset_0_0_0_1px_var(--border-strong)]',
				light ? lightBackground : darkBackground
			)}
		>
			{#if src && !brokenLogos.has(src)}
				<img
					{src}
					onerror={() => brokenLogos.add(src)}
					alt={light ? m.name_logo({ name: clientName }) : m.name_dark_logo({ name: clientName })}
					class={cn('size-14 object-contain', !light && !darkLogoDataURL && 'opacity-50')}
				/>
			{:else}
				<span aria-hidden="true" class="text-[22px] font-medium">{monogram(clientName)}</span>
			{/if}
		</span>
		<span class="text-caption text-fg-muted text-center whitespace-nowrap">
			{light ? m.light() : m.dark()}
			{#if !light && !darkLogoDataURL && logoDataURL}
				<span class="text-micro block">{m.using_light_logo()}</span>
			{/if}
		</span>
	</div>
{/snippet}

<div role="group" aria-labelledby="oidc-client-logo-label" class="flex flex-col gap-1.5">
	<span id="oidc-client-logo-label" class="text-control font-semibold">{m.logo()}</span>
	<p class="text-caption text-fg-muted">{m.oidc_client_logo_spec()}</p>
	<div class="mt-1 flex flex-wrap items-start gap-x-4 gap-y-3">
		<div class="flex gap-2">
			{@render well(effectiveLightLogoURL, true)}
			{@render well(effectiveDarkLogoURL, false)}
		</div>
		<div class="flex flex-col items-start gap-2">
			<Popover.Root bind:open {onOpenChange}>
				<Popover.Trigger bind:ref={triggerRef}>
					{#snippet child({ props })}
						<Button {...props} variant="secondary" size="sm">
							<LucideUpload />
							{m.choose_logo()}
						</Button>
					{/snippet}
				</Popover.Trigger>
				<Popover.Content
					bind:ref={contentRef}
					class="w-[min(24rem,calc(100vw-2rem))] gap-0 overflow-hidden p-0"
					align="start"
					sideOffset={popoverOffset}
					{alignOffset}
					avoidCollisions={false}
					strategy="fixed"
				>
					<!-- The popover isn't positioned yet when the first icon gets selected, so scrolling it into view would scroll the page instead -->
					<Command.Root shouldFilter={false} columns={3} disableInitialScroll class="rounded-none">
						<Command.Input
							placeholder={presetsEnabled ? m.search_icons_or_paste_url() : m.paste_image_url()}
							aria-label={presetsEnabled ? m.search_icons_or_paste_url() : m.paste_image_url()}
							bind:value={search}
							oninput={(e) => onSearch(e.currentTarget.value)}
							onkeydown={onInputKeydown}
						/>
						<!-- Without icons the list shrinks to fit the remaining tiles -->
						<Command.List class={cn('px-2 pt-4 pb-2', showIconGrid && 'h-80')}>
							<div class="grid grid-cols-3 gap-2">
								<!-- The first tile adds a custom image, either from a pasted URL or as an upload -->
								<div class="grid h-14 grid-cols-2 overflow-hidden rounded-sm">
									{#if imageUrl}
										{@render urlHalf(imageUrl, true)}
										{@render urlHalf(imageUrl, false)}
									{:else}
										{@render uploadHalf(true)}
										{@render uploadHalf(false)}
									{/if}
								</div>
								{#if !imageUrl}
									<!-- The current logo gets its own tile, even before it is saved, because a picked icon isn't necessarily in the search results -->
									<!-- As the first item of the list it is what the picker selects, so it shows which logo is in use -->
									{#if hasLogo}
										<CommandPrimitive.Item
											value="custom"
											onSelect={() => (open = false)}
											aria-label={m.current_logo()}
											title={m.current_logo()}
											class="data-selected:ring-ring grid h-14 cursor-pointer grid-cols-2 overflow-hidden rounded-sm outline-hidden select-none data-selected:ring-2"
										>
											{@render tileHalf(effectiveLightLogoURL, true)}
											{@render tileHalf(effectiveDarkLogoURL, false)}
										</CommandPrimitive.Item>
									{/if}
									{#if presetsEnabled}
										{#each presets as preset (preset.reference)}
											<CommandPrimitive.Item
												value={preset.reference}
												onSelect={() => selectPreset(preset)}
												aria-label={preset.name}
												title={preset.name}
												class="data-selected:ring-ring grid h-14 cursor-pointer grid-cols-2 overflow-hidden rounded-sm outline-hidden select-none data-selected:ring-2"
											>
												{@render tileHalf(preset.logoUrl, true)}
												{@render tileHalf(preset.darkLogoUrl ?? preset.logoUrl, false)}
											</CommandPrimitive.Item>
										{/each}
									{/if}
								{/if}
							</div>
							{#if showPresets && presets.length === 0 && presetsQuery !== null}
								<div class="text-fg-muted text-control flex justify-center px-3 py-6 text-center">
									{errorMessage ?? m.no_matches_for_query({ query: presetsQuery.trim() })}
								</div>
							{/if}
						</Command.List>
					</Command.Root>
					<!-- A custom icon library isn't necessarily selfh.st's collection, so the credit only applies to the default one while its icons are shown -->
					{#if iconLibrary === 'default' && showIconGrid}
						<p class="text-fg-muted text-micro border-t px-3 py-2">
							<FormattedMessage message={m.icons_provided_by_selfhst} />
						</p>
					{/if}
				</Popover.Content>
			</Popover.Root>
			<Button variant="ghost" size="sm" disabled={!hasLogo} onclick={onReset}>
				{m.remove_logo()}
			</Button>
		</div>
	</div>
</div>
