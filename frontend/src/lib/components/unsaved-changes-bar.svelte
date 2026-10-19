<script lang="ts">
	import { modKey } from '#lib/utils/keyboard-util.ts';
	import { beforeNavigate } from '$app/navigation';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Tooltip from '#lib/components/ui/tooltip/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import unsavedChanges from '#lib/stores/unsaved-changes-store.svelte.ts';
	import { cn } from '#lib/utils/style.ts';
	import { LucideCircleAlert, LucideCircleCheck } from '@lucide/svelte';
	import { tick } from 'svelte';
	import { cubicOut } from 'svelte/easing';
	import { prefersReducedMotion } from 'svelte/motion';

	const ROW_HEIGHT = 52;
	const VIEWPORT_MARGIN = 16;
	const INSET = 16;
	const INSET_BUTTONS = 8;

	const MOTION = 'duration-[380ms] ease-[cubic-bezier(0.22,1,0.36,1)]';
	const LAYER = 'absolute top-0 flex items-center whitespace-nowrap';
	const SLOT_GLYPH = 'col-start-1 row-start-1 size-3.5 transition-opacity duration-(--dur-fast)';

	type BloomTint = 'pending' | 'success' | 'error';

	let shake = $state(false);
	let blocked = $state(false);
	let bloom = $state<{ tint: BloomTint }>();
	let barWidth = $state<number>();
	let compact = $state(false);
	let column = $state<{ left: number; width: number }>();
	let frame = $state<HTMLElement>();
	let pendingMessage = $state<HTMLElement>();
	let statusMessage = $state<HTMLElement>();
	let actions = $state<HTMLElement>();

	const animate = $derived(!$appConfigStore?.disableAnimations && !prefersReducedMotion.current);
	const status = $derived(unsavedChanges.status);
	const showingStatus = $derived(!!status);
	const showingActions = $derived(unsavedChanges.hasChanges);
	const barOpen = $derived(showingActions || showingStatus);
	const pendingText = $derived(
		unsavedChanges.dirtyCount === 1
			? m.one_unsaved_change()
			: m.count_unsaved_changes({ count: unsavedChanges.dirtyCount })
	);
	const announcement = $derived(
		status?.message ??
			(blocked ? m.save_or_discard_before_leaving() : showingActions ? pendingText : '')
	);

	// The status line keeps its last message while it slides away, so the bar does not shrink under it
	let shownStatus = $state<typeof status>(null);
	$effect(() => {
		if (status) shownStatus = status;
	});

	// Settled means in view; the other line waits a row height away
	const pendingSettled = $derived(!showingStatus && !compact);
	const statusSettled = $derived(showingStatus && !compact);

	// A new object every time, so {#key} remounts the layer and replays the animation
	function flashBloom(tint: BloomTint) {
		bloom = { tint };
	}

	$effect(() => {
		if (barOpen) flashBloom('pending');
	});

	$effect(() => {
		if (status) flashBloom(status.type === 'error' ? 'error' : 'success');
	});

	// The bar is mounted outside the shell, so it follows the page's <main> to center on the content column rather than the window
	function measureColumn() {
		const rect = document.querySelector('main')?.getBoundingClientRect();
		column = rect && { left: rect.left, width: rect.width };
	}

	function measure() {
		const message = showingStatus ? statusMessage : pendingMessage;
		if (!message || !actions || !frame) return;

		// The line carries its own trailing inset; offsetWidth ignores the scale the buttons carry while fading, and rounding the text up keeps its last glyph
		const end = showingActions ? actions.offsetWidth + INSET_BUTTONS : 0;
		const roomy = INSET + Math.ceil(message.getBoundingClientRect().width) + end;

		// An unmeasured frame reports no width, which must not read as "nothing fits"
		const available = frame.clientWidth ? frame.clientWidth - 2 * VIEWPORT_MARGIN : Infinity;

		compact = showingActions && roomy > available;
		barWidth = Math.min(compact ? actions.offsetWidth + 2 * INSET_BUTTONS : roomy, available);
	}

	$effect(measure);

	// Catches what no state change announces: a longer message, a font loading, the viewport resizing, the sidebar collapsing
	$effect(() => {
		if (!barOpen) return;
		const main = document.querySelector('main');
		const layers = [pendingMessage, statusMessage, actions, main, main?.parentElement].filter(
			(layer) => !!layer
		);

		const observer = new ResizeObserver(() => {
			measureColumn();
			measure();
		});
		for (const layer of layers) observer.observe(layer);

		return () => observer.disconnect();
	});

	function rise(_node: HTMLElement, { duration = 220 } = {}) {
		return {
			duration: animate ? duration : 0,
			easing: cubicOut,
			css: (t: number, u: number) =>
				`opacity: ${t}; transform: translateY(${u * 8}px) scale(${0.98 + t * 0.02})`
		};
	}

	beforeNavigate((nav) => {
		if (nav.shallow || !unsavedChanges.hasChanges) return;

		// Cancelling a full page unload makes the browser show its own "leave site?" prompt.
		nav.cancel();
		if (nav.type === 'leave') return;

		shake = true;
		blocked = true;
		flashBloom('pending');
	});

	// A fresh edit supersedes any status message left over from a previous save.
	$effect(() => {
		if (unsavedChanges.hasChanges) unsavedChanges.clearStatus();
		else blocked = false;
	});

	function handleSaveShortcut(event: KeyboardEvent) {
		if (event.key.toLowerCase() !== 's' || !(event.metaKey || event.ctrlKey)) return;
		if (!unsavedChanges.hasChanges || unsavedChanges.saving) return;

		event.preventDefault();
		void saveAll();
	}

	async function saveAll() {
		if ((await unsavedChanges.saveAll()) === 'invalid') await revealFirstInvalidField();
	}

	// Brings the first field that failed validation into view, switching to the tab that contains
	// it if necessary, since the bar is global while the field may sit on a hidden tab.
	async function revealFirstInvalidField() {
		await tick();
		const field = document.querySelector<HTMLElement>(
			'[aria-invalid="true"], [data-slot="field-error"]'
		);
		if (!field) return;

		for (
			let panel = field.closest<HTMLElement>('[role="tabpanel"]');
			panel;
			panel = panel.parentElement?.closest<HTMLElement>('[role="tabpanel"]') ?? null
		) {
			const value = panel.dataset.value;
			if (value === undefined) continue;
			panel
				.closest('[data-slot="tabs"]')
				?.querySelector<HTMLElement>(`[role="tab"][data-value="${CSS.escape(value)}"]`)
				?.click();
		}

		await tick();
		field.scrollIntoView({ block: 'center', behavior: animate ? 'smooth' : 'instant' });
		field.focus({ preventScroll: true });
	}
</script>

<svelte:window onkeydown={handleSaveShortcut} />

<p class="sr-only" aria-live="polite">{announcement}</p>

{#if barOpen}
	<div
		bind:this={frame}
		class={cn(
			'fixed bottom-5 z-(--z-bar) flex max-sm:bottom-[calc(76px+env(safe-area-inset-bottom))]',
			compact ? 'justify-end' : 'justify-center'
		)}
		style:left="{column?.left ?? 0}px"
		style:width={column ? `${column.width}px` : '100%'}
		style:padding-inline="{VIEWPORT_MARGIN}px"
		transition:rise
		class:animate-shake={shake}
		onanimationend={() => (shake = false)}
	>
		<div
			role="region"
			aria-label={m.unsaved_changes()}
			class={cn(
				'bg-inverse text-inverse-fg shadow-overlay inset-ring-inverse-border relative isolate max-w-full overflow-hidden rounded-lg inset-ring',
				animate && `transition-[width] ${MOTION}`
			)}
			style:height="{ROW_HEIGHT}px"
			style:width={barWidth ? `${barWidth}px` : undefined}
		>
			{#if animate && bloom}
				{#key bloom}
					<span
						aria-hidden="true"
						class="animate-bloom pointer-events-none absolute inset-0 -z-1 bloom-{bloom.tint}"
					></span>
				{/key}
			{/if}

			<span
				bind:this={pendingMessage}
				class={cn(LAYER, 'gap-2.5 pr-4', animate && `transition-[translate] ${MOTION}`)}
				style:height="{ROW_HEIGHT}px"
				style:left="{INSET}px"
				style:translate={pendingSettled ? '0' : `0 -${ROW_HEIGHT}px`}
				inert={!pendingSettled}
			>
				<span aria-hidden="true" class="grid size-3.5 place-items-center">
					<span class="bg-warning-solid size-2 rounded-full"></span>
				</span>
				<span class="text-control font-medium tabular-nums">{pendingText}</span>
			</span>

			<span
				bind:this={statusMessage}
				class={cn(LAYER, 'gap-2.5 pr-4', animate && `transition-[translate] ${MOTION}`)}
				style:height="{ROW_HEIGHT}px"
				style:left="{INSET}px"
				style:translate={statusSettled ? '0' : `0 ${ROW_HEIGHT}px`}
				inert={!statusSettled}
			>
				<span aria-hidden="true" class="grid">
					<LucideCircleCheck
						class={cn(
							SLOT_GLYPH,
							'text-success-solid',
							shownStatus?.type !== 'success' && 'opacity-0'
						)}
					/>
					<LucideCircleAlert
						class={cn(
							SLOT_GLYPH,
							'text-danger-solid',
							shownStatus?.type !== 'error' && 'opacity-0'
						)}
					/>
				</span>
				<span class="text-control font-medium">{shownStatus?.message ?? ''}</span>
			</span>

			<span
				bind:this={actions}
				class={cn(
					LAYER,
					'gap-1',
					animate && `transition-[opacity,scale] ${MOTION}`,
					!showingActions && 'scale-[0.96] opacity-0'
				)}
				style:height="{ROW_HEIGHT}px"
				style:right="{INSET_BUTTONS}px"
				inert={!showingActions}
			>
				<Button
					variant="ghost"
					size="sm"
					class="text-inverse-fg before:bg-inverse-hover focus-visible:outline-inverse-fg"
					disabled={unsavedChanges.saving}
					onclick={() => unsavedChanges.discardAll()}
				>
					{m.discard()}
				</Button>
				<!-- The bar lives outside the settings shell's tooltip provider -->
				<Tooltip.Provider>
					<Tooltip.Root>
						<Tooltip.Trigger>
							{#snippet child({ props })}
								<Button
									{...props}
									variant="inverse"
									size="sm"
									isLoading={unsavedChanges.saving}
									onclick={saveAll}
								>
									{m.save()}
								</Button>
							{/snippet}
						</Tooltip.Trigger>
						<Tooltip.Content>
							{m.save()}
							<kbd class="text-inverse-muted font-sans">{modKey}S</kbd>
						</Tooltip.Content>
					</Tooltip.Root>
				</Tooltip.Provider>
			</span>
		</div>
	</div>
{/if}
