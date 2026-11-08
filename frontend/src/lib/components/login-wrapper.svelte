<script module lang="ts">
	let backgroundImageExists = $state<boolean | undefined>(undefined);

	// Where the card was when the last auth page left, so the next step grows or slides from there instead of jumping
	let previousCard: { rect: DOMRect; at: number } | undefined;
</script>

<script lang="ts">
	import { beforeNavigate, onNavigate } from '$app/navigation';
	import StaySignedInStep from '#lib/components/auth/stay-signed-in-step.svelte';
	import { Button } from '@pocket-id/ui/button';
	import { ScrollArea } from '@pocket-id/ui/scroll-area';
	import { m } from '#lib/paraglide/messages.js';
	import { appReveal } from '#lib/stores/app-reveal-store.svelte.ts';
	import { keepSignedIn } from '#lib/stores/keep-signed-in-store.svelte.ts';
	import { cachedBackgroundImage } from '#lib/utils/cached-asset-util.ts';
	import { cn } from '@pocket-id/ui/utils';
	import { cssMs, cssVar } from '#lib/utils/style.ts';
	import ChevronLeftIcon from '~icons/solar/alt-arrow-left-linear';
	import type { Snippet } from 'svelte';
	import { prefersReducedMotion } from 'svelte/motion';
	import { MediaQuery } from 'svelte/reactivity';

	let {
		children,
		side,
		back: backHref,
		wide = false
	}: {
		children: Snippet;
		// Follows the children on narrow steps and fills the second column of a wide one, next to the children that stay in view while it scrolls
		side?: Snippet;
		back?: string;
		// Steps with scopes or technical details grow the desktop card
		wide?: boolean;
	} = $props();

	// Going back from the question would leave a sign-in that already happened
	const back = $derived(keepSignedIn.asking ? undefined : backHref);

	const isCard = new MediaQuery('(min-width: 640px)');
	// Matches the lg breakpoint where a wide card splits into two columns
	const isTwoColumn = new MediaQuery('(min-width: 1024px)');
	const photo = $derived(backgroundImageExists !== false);

	let card = $state<HTMLDivElement>();
	let column = $state<HTMLDivElement>();
	let stack = $state<HTMLDivElement>();
	let surface = $state<HTMLDivElement>();
	// The strip starts at the previous step's height so it follows the sheet instead of zooming out from full height
	let cardHeight = $state(previousCard?.rect.height ?? 0);
	let innerHeight = $state(0);
	const fullHeight = $derived(cardHeight >= innerHeight - 1);
	let photoLoaded = $state(backgroundImageExists === true);

	let last: { rect: DOMRect; width: number; height: number } | undefined;
	let resizing = $state.raw<Animation>();
	let contentBelow = $state(false);

	beforeNavigate(() => {
		if (card) previousCard = { rect: card.getBoundingClientRect(), at: performance.now() };
	});

	// Leaving sign-up for the app unfolds the card into the app's surface, and the settings shell then fades in on it
	// onNavigate runs once the app's data has loaded, so the app swaps in the moment the card covers the screen
	onNavigate((navigation) => {
		if (
			!card ||
			!column ||
			!surface ||
			!navigation.from?.route.id?.startsWith('/signup') ||
			!navigation.to?.route.id?.startsWith('/settings')
		) {
			return;
		}
		appReveal.pending = true;
		return unfold(card, column, surface);
	});

	async function unfold(card: HTMLDivElement, column: HTMLDivElement, surface: HTMLDivElement) {
		await column.animate([{ opacity: 1 }, { opacity: 0 }], {
			duration: cssMs('--dur-fast'),
			easing: cssVar('--ease-in'),
			fill: 'forwards'
		}).finished;

		// A sheet that already fills the phone screen has nothing left to unfold
		const from = card.getBoundingClientRect();
		if (from.width >= innerWidth - 1 && from.height >= innerHeight - 1) return;

		// A separate surface grows over the card, so the card's own resize handling never sees the change
		surface.hidden = false;
		await surface.animate(
			[
				{
					left: `${from.left}px`,
					top: `${from.top}px`,
					width: `${from.width}px`,
					height: `${from.height}px`,
					borderRadius: getComputedStyle(card).borderRadius
				},
				{ left: '0px', top: '0px', width: '100%', height: '100%', borderRadius: '0px' }
			],
			{ duration: cssMs('--dur-unfold'), easing: cssVar('--ease-in-out'), fill: 'forwards' }
		).finished;
	}

	// The stack keeps its final layout while the card animates, so a step that changes mid-animation is still noticed
	$effect(() => {
		if (!card || !column || !stack) return;
		const observer = new ResizeObserver(() => onResize(card!, column!));
		observer.observe(stack);
		return () => observer.disconnect();
	});

	function onResize(card: HTMLDivElement, column: HTMLDivElement) {
		// An interrupted animation continues from where the card is drawn right now
		// The new layout already applies here, so the sheet is drawn at its previous layout shifted by the running translate
		let drawn: DOMRect | undefined;
		if (resizing && last && !isCard.current) {
			const dy = parseFloat(getComputedStyle(card).translate.split(' ')[1] ?? '0');
			drawn = new DOMRect(last.rect.x, last.rect.y + dy, last.rect.width, last.rect.height);
		} else if (resizing) {
			drawn = card.getBoundingClientRect();
		}
		settleResize(column);
		cardHeight = card.offsetHeight;
		updateContentBelow();

		// A new page starts from the previous page's card when it left a moment ago
		const entering = !last;
		const from =
			drawn ??
			last?.rect ??
			(previousCard && performance.now() - previousCard.at < 5000 ? previousCard.rect : undefined);
		previousCard = undefined;

		const windowResized = last && (last.width !== innerWidth || last.height !== innerHeight);
		const to = card.getBoundingClientRect();
		last = { rect: to, width: innerWidth, height: innerHeight };
		if (!from || windowResized || prefersReducedMotion.current) return;

		if (isCard.current) growCard(card, column, from, to, entering || to.width > from.width);
		else if (photo) slideSheet(card, column, from.top - to.top, entering);
		else if (entering) fadeIn(column, 0);
	}

	// The content keeps its final size throughout, so the card's edge reveals it instead of it reflowing on every frame
	function growCard(
		card: HTMLDivElement,
		column: HTMLDivElement,
		from: DOMRect,
		to: DOMRect,
		newStep: boolean
	) {
		const sizeChanged =
			Math.abs(to.width - from.width) >= 1 || Math.abs(to.height - from.height) >= 1;
		if (!sizeChanged) {
			if (newStep) fadeIn(column, 0);
			return;
		}

		column.style.width = `${to.width}px`;
		column.style.height = `${to.height}px`;
		column.style.placeSelf = 'center';
		resizing = card.animate(
			[
				{ width: `${from.width}px`, height: `${from.height}px` },
				{ width: `${to.width}px`, height: `${to.height}px` }
			],
			{ duration: cssMs('--dur-slow'), easing: cssVar('--ease-in-out') }
		);
		resizing.onfinish = () => settleResize(column);

		// A new step fades in once the card is halfway open, while a height change or a retract inside a step only moves the card's edges so the outcome's own animation stays visible
		if (newStep) fadeIn(column, cssMs('--dur-slow') / 2);
	}

	// The sheet's top edge travels from where it was, and a surface extension below it covers the photo while it moves up
	function slideSheet(card: HTMLDivElement, column: HTMLDivElement, dy: number, entering: boolean) {
		if (Math.abs(dy) >= 0.5) {
			resizing = card.animate([{ translate: `0 ${dy}px` }, { translate: '0 0' }], {
				duration: cssMs('--dur-slow'),
				easing: cssVar('--ease-in-out')
			});
			resizing.onfinish = () => settleResize(column);
		}
		if (entering) fadeIn(column, 0);
	}

	function fadeIn(column: HTMLDivElement, delay: number) {
		column.animate([{ opacity: 0 }, { opacity: 1 }], {
			duration: cssMs('--dur-base'),
			delay,
			easing: cssVar('--ease-out'),
			fill: 'backwards'
		});
	}

	function settleResize(column: HTMLDivElement) {
		resizing?.cancel();
		resizing = undefined;
		column.style.width = '';
		column.style.height = '';
		column.style.placeSelf = '';
	}

	function updateContentBelow() {
		if (column) contentBelow = column.scrollHeight - column.scrollTop - column.clientHeight > 1;
	}
</script>

<svelte:window bind:innerHeight onresize={updateContentBelow} />

<!-- The placeholder fill holds the photo's place until it loads -->
<div
	class={cn('sm:bg-sidebar fixed inset-0 isolate overflow-hidden', photo && 'max-sm:bg-fill-muted')}
>
	{#if photo}
		<!-- On phones the photo fills the strip above the full-width sheet and follows its top edge, on wider screens it is the backdrop behind the card -->
		<!-- Its height only transitions while the sheet slides, so a page that opens without a slide does not zoom the photo -->
		<img
			src={cachedBackgroundImage.getUrl()}
			alt=""
			class={cn(
				'pointer-events-none absolute inset-x-0 top-0 -z-1 h-(--strip) w-full object-cover opacity-0 transition-opacity duration-(--dur-slow) ease-in-out sm:h-full',
				resizing && 'transition-[opacity,height]',
				photoLoaded && 'opacity-100'
			)}
			style:--strip="{fullHeight ? 0 : innerHeight - cardHeight + 12}px"
			onload={() => (backgroundImageExists = photoLoaded = true)}
			onerror={() => (backgroundImageExists = false)}
		/>
	{/if}

	<div
		bind:this={card}
		class={cn(
			'grid grid-rows-[minmax(0,1fr)]',
			'sm:bg-surface sm:inset-ring-border sm:absolute sm:inset-0 sm:m-auto sm:h-fit sm:max-h-[calc(100%-64px)] sm:w-[min(520px,100%-64px)] sm:overflow-hidden sm:rounded-2xl sm:inset-ring',
			photo && 'sm:shadow-overlay',
			wide && 'lg:w-[min(1040px,100%-64px)]',
			photo
				? 'max-sm:bg-surface max-sm:shadow-overlay max-sm:after:bg-surface max-sm:absolute max-sm:inset-x-0 max-sm:bottom-0 max-sm:max-h-full max-sm:rounded-t-md max-sm:after:absolute max-sm:after:inset-x-0 max-sm:after:top-full max-sm:after:h-dvh'
				: 'max-sm:h-full',
			photo && fullHeight && 'max-sm:rounded-none'
		)}
	>
		<!-- The scroll padding keeps keyboard focus clear of the sticky action panel on phones, sized for its two buttons and a footnote of up to two lines -->
		<!-- Without the photo the side padding itself centres the stack, so the action panel reaches the page edges through it -->
		<div
			bind:this={column}
			class={cn(
				'group/sheet flex min-h-0 flex-col overflow-y-auto max-sm:px-(--sheet-x) max-sm:pb-(--sheet-b) max-sm:has-data-action-panel:scroll-pb-[calc(176px+var(--sheet-b))] sm:px-12',
				photo
					? '[--sheet-b:calc(24px+env(safe-area-inset-bottom))] [--sheet-x:24px] max-[360px]:[--sheet-x:16px] max-sm:pt-6'
					: '[--sheet-b:24px] [--sheet-x:max(16px,calc((100vw_-_var(--col-auth))/2))]'
			)}
			data-content-below={contentBelow || undefined}
			onscroll={updateContentBelow}
		>
			<div
				class={cn(
					'mx-auto flex w-full shrink-0 items-center sm:min-h-14 sm:pt-4',
					photo
						? 'max-sm:-mt-3 max-sm:min-h-11'
						: 'max-sm:min-h-17 max-sm:max-w-(--col-auth) max-sm:pt-6',
					photo && !back && 'max-sm:hidden',
					!wide && 'sm:max-w-(--col-auth)'
				)}
			>
				{#if back}
					<Button
						variant="link"
						href={back}
						class="text-body text-fg-muted hover:text-fg -ml-1 gap-1"
					>
						<ChevronLeftIcon />
						{m.back()}
					</Button>
				{/if}
			</div>

			<!-- In two columns the stack shrinks with the card so the side column scrolls on its own instead of the whole card -->
			<main
				class={cn(
					'flex flex-[1_0_auto] items-center justify-center sm:pt-4 sm:pb-8',
					wide && side && 'lg:min-h-0 lg:shrink'
				)}
			>
				<div
					bind:this={stack}
					class={cn(
						'flex w-full flex-col',
						photo ? 'sm:max-w-(--col-auth)' : 'max-w-(--col-auth)',
						wide && 'lg:max-w-none',
						wide &&
							side &&
							'lg:before:bg-border lg:relative lg:grid lg:h-full lg:grid-cols-2 lg:grid-rows-[minmax(0,1fr)] lg:gap-x-24 lg:before:absolute lg:before:inset-y-0 lg:before:left-1/2 lg:before:w-px'
					)}
				>
					<!-- Every sign-in page asks in its own card, so the question sits here instead of in each page -->
					{#if keepSignedIn.asking}
						<StaySignedInStep />
					{:else if side}
						<!-- The columns dissolve outside the wide layout, so the phone action panel still sticks to the whole sheet -->
						<div class={cn('contents', wide && 'lg:flex lg:flex-col')}>
							{@render children()}
						</div>
						{#if wide && isTwoColumn.current}
							<!-- The padding gives focus rings room inside the scroll clip and puts the scrollbar beside the content instead of over it -->
							<!-- A positioned viewport keeps absolutely positioned descendants such as sr-only labels inside the scroll area, otherwise they overflow the stack and scroll the whole card -->
							<ScrollArea class="-m-1 -mr-4" viewportClass="relative p-1 pr-4">
								{@render side()}
							</ScrollArea>
						{:else}
							{@render side()}
						{/if}
					{:else}
						{@render children()}
					{/if}
				</div>
			</main>

			<!-- Balances the top bar so the stack stays centered in the desktop card -->
			<!-- On short windows it gives way before the card has to scroll -->
			<div class="max-sm:hidden sm:h-16 sm:min-h-4"></div>
		</div>
	</div>

	<div bind:this={surface} class="bg-surface absolute" hidden></div>
</div>
