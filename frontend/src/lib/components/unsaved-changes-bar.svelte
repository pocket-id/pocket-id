<script lang="ts">
	import { beforeNavigate } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import unsavedChanges from '$lib/stores/unsaved-changes-store.svelte';
	import { cn } from '$lib/utils/style';
	import { LucideCircleAlert } from '@lucide/svelte';
	import { tick } from 'svelte';
	import { cubicOut } from 'svelte/easing';

	const ROW_HEIGHT = 44;
	const BOTTOM_OFFSET = 20;
	const VIEWPORT_MARGIN = 16;
	const INSET = 18;
	const INSET_BUTTONS = 8;

	const MOTION =
		'duration-[380ms] ease-[cubic-bezier(0.22,1,0.36,1)] motion-reduce:transition-none';
	const LAYER = 'absolute top-0 flex items-center whitespace-nowrap';

	type BloomTint = 'pending' | 'success' | 'error';

	let shake = $state(false);
	let bloom = $state<{ tint: BloomTint }>();
	let pillWidth = $state<number>();
	let compact = $state(false);
	let frame = $state<HTMLElement>();
	let pendingMessage = $state<HTMLElement>();
	let statusMessage = $state<HTMLElement>();
	let actions = $state<HTMLElement>();

	const animate = $derived(!$appConfigStore?.disableAnimations);
	const status = $derived(unsavedChanges.status);
	const showingStatus = $derived(!!status);
	const showingActions = $derived(unsavedChanges.hasChanges);
	const barOpen = $derived(showingActions || showingStatus);

	// Settled means in view; the other message waits a row height away
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

	function measure() {
		const message = showingStatus ? statusMessage : pendingMessage;
		if (!message || !actions || !frame) return;

		// offsetWidth ignores the scale the buttons carry while fading, and rounding the text up keeps its last glyph
		const end = showingActions ? actions.offsetWidth + INSET_BUTTONS : INSET;
		const roomy = INSET + Math.ceil(message.getBoundingClientRect().width) + end;

		// An unmeasured frame reports no width, which must not read as "nothing fits"
		const available = frame.clientWidth ? frame.clientWidth - 2 * VIEWPORT_MARGIN : Infinity;

		compact = showingActions && roomy > available;
		pillWidth = Math.min(compact ? actions.offsetWidth + 2 * INSET_BUTTONS : roomy, available);
	}

	$effect(measure);

	// Catches what no state change announces: a longer message, a font loading, the viewport resizing
	$effect(() => {
		const layers = [pendingMessage, statusMessage, actions, frame].filter((layer) => !!layer);
		if (layers.length === 0) return;

		const observer = new ResizeObserver(measure);
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
		if (!unsavedChanges.hasChanges) return;

		// Cancelling a full page unload makes the browser show its own "leave site?" prompt.
		nav.cancel();
		if (nav.type === 'leave') return;

		shake = true;
		flashBloom('pending');
	});

	// A fresh edit supersedes any status message left over from a previous save.
	$effect(() => {
		if (unsavedChanges.hasChanges) unsavedChanges.clearStatus();
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
		field.scrollIntoView({ block: 'center', behavior: 'smooth' });
		field.focus({ preventScroll: true });
	}
</script>

<svelte:window onkeydown={handleSaveShortcut} />

<!-- Belongs to the message, so it leaves with it and never counts towards the buttons' width -->
{#snippet divider()}
	<span class="bg-border ml-2.5 mr-2 h-4.5 w-px shrink-0"></span>
{/snippet}

{#if barOpen}
	<div
		bind:this={frame}
		class={cn('fixed inset-x-0 z-50 flex', compact ? 'justify-end' : 'justify-center')}
		style:bottom="{BOTTOM_OFFSET}px"
		style:padding-inline="{VIEWPORT_MARGIN}px"
		transition:rise
		class:animate-shake={shake}
		onanimationend={() => (shake = false)}
	>
		<div
			role="status"
			aria-live="polite"
			class={cn(
				'bg-popover text-popover-foreground shadow-raised dark:bg-secondary relative isolate max-w-full overflow-hidden rounded-full border',
				animate && `transition-[width] ${MOTION}`
			)}
			style:height="{ROW_HEIGHT}px"
			style:width={pillWidth ? `${pillWidth}px` : undefined}
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
				class={cn(LAYER, 'gap-2.5', animate && `transition-[translate] ${MOTION}`)}
				style:height="{ROW_HEIGHT}px"
				style:left="{INSET}px"
				style:translate={pendingSettled ? '0' : `0 -${ROW_HEIGHT}px`}
				inert={!pendingSettled}
			>
				{#if unsavedChanges.dirtyCount > 1}
					<span
						class="bg-primary/15 text-foreground flex size-5 items-center justify-center rounded-full font-bold tabular-nums"
					>
						{unsavedChanges.dirtyCount}
					</span>
				{/if}
				<span class="text-sm font-semibold tracking-tight">{m.you_have_unsaved_changes()}</span>
				{@render divider()}
			</span>

			<span
				bind:this={statusMessage}
				class={cn(LAYER, 'gap-2.5', animate && `transition-[translate] ${MOTION}`)}
				style:height="{ROW_HEIGHT}px"
				style:left="{INSET}px"
				style:translate={statusSettled ? '0' : `0 ${ROW_HEIGHT}px`}
				inert={!statusSettled}
			>
				{#if status?.type === 'error'}
					<LucideCircleAlert class="text-destructive size-4 shrink-0" />
				{/if}
				<span class="text-sm font-semibold tracking-tight">{status?.message ?? ''}</span>
				{#if showingActions}
					{@render divider()}
				{/if}
			</span>

			<span
				bind:this={actions}
				class={cn(
					LAYER,
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
					class="hover:bg-foreground/10 dark:hover:bg-foreground/10"
					disabled={unsavedChanges.saving}
					onclick={() => unsavedChanges.discardAll()}
				>
					{m.discard()}
				</Button>
				<Button class="ml-1.5" size="sm" isLoading={unsavedChanges.saving} onclick={saveAll}>
					{m.save()}
				</Button>
			</span>
		</div>
	</div>
{/if}
