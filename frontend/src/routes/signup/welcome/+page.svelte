<script lang="ts">
	import ActionPanel from '#lib/components/auth/action-panel.svelte';
	import AuthHeading from '#lib/components/auth/auth-heading.svelte';
	import AuthTile from '#lib/components/auth/auth-tile.svelte';
	import { authPrimaryClass } from '#lib/components/auth/auth-util.ts';
	import SignInWrapper from '#lib/components/login-wrapper.svelte';
	import { Button } from '@pocket-id/ui/button';
	import { m } from '#lib/paraglide/messages.js';
	import AppConfigService from '#lib/services/app-config-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { Feature } from '#lib/types/feature.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { featureSections } from '#lib/utils/feature-util.ts';
	import { cn } from '@pocket-id/ui/utils';
	import { goto } from '$app/navigation';
	import CheckCircleIcon from '~icons/solar/check-circle-bold';

	let { data } = $props();

	const appConfigService = new AppConfigService();
	const uid = $props.id();
	const sections = $derived(featureSections($appConfigStore.appName));

	let selected = $state(
		(Object.keys(data.features) as Feature[]).filter((id) => data.features[id].enabled)
	);
	let isLoading = $state(false);
	let grid = $state<HTMLDivElement>();
	let innerWidth = $state(0);
	let gridMinHeight = $state(0);
	let slots = $state<number[]>([]);
	// Titles that wrap in longer languages change the heights once the web fonts arrive
	let fontsLoaded = $state(false);
	document.fonts.ready.then(() => (fontsLoaded = true));

	// One description is open at a time and stays open until another one opens, so its room only ever moves between tiles
	let active = $state<Feature>();
	let timer: ReturnType<typeof setTimeout>;
	$effect(() => () => clearTimeout(timer));

	// Below this speed in pixels per millisecond the pointer is aiming, which is about one tile in a quarter of a second
	const SLOW = 0.2;
	// The speed is taken over at least this many milliseconds, because single events of a fast mouse lie a pixel apart at any speed
	const WINDOW = 40;
	// A resting pointer sends no events, so this many milliseconds without one mean it stopped
	const STOPPED = 60;
	let sample = { x: 0, y: 0, time: 0 };

	// Every description in a column opens to the height of the column's tallest one
	// The tile under the pointer then always stays under it: it grows upward when the open tile is above it and downward otherwise
	// The grid keeps room for the tallest column with one description open, so the card never changes size
	$effect(() => {
		void innerWidth;
		void fontsLoaded;
		if (!grid) return;
		const columns = ([...grid.children] as HTMLElement[]).map((column) => {
			const descriptions = [...column.querySelectorAll<HTMLElement>('[data-description]')];
			const slot = Math.max(0, ...descriptions.map((d) => d.scrollHeight));
			// Fractional tile heights would otherwise leave the grid a fraction short and the card would twitch
			const height = (element: Element) => element.getBoundingClientRect().height;
			const open = descriptions.reduce((sum, d) => sum + height(d.parentElement!), 0);
			return { slot, height: Math.ceil(height(column) - open + slot) };
		});
		slots = columns.map((column) => column.slot);
		gridMinHeight = Math.max(0, ...columns.map((column) => column.height));
	});

	// A pointer that slows down on a tile or stops on it opens the description, while one that crosses the tile fast opens nothing
	function point(event: PointerEvent, id: Feature) {
		if (event.pointerType === 'touch') return;

		// Measure the speed since the last sample, also over the open tile, so the first event on the next tile has a fresh one
		// A sample from before a rest says nothing about the speed now, so it only starts a new measurement
		const elapsed = event.timeStamp - sample.time;
		let slow = false;
		if (elapsed >= WINDOW) {
			const distance = Math.hypot(event.clientX - sample.x, event.clientY - sample.y);
			slow = elapsed < WINDOW + STOPPED && distance / elapsed < SLOW;
			sample = { x: event.clientX, y: event.clientY, time: event.timeStamp };
		}
		if (id === active) return;

		// Every event of a moving pointer pushes the stop back, so only a pointer that came to rest reaches it
		clearTimeout(timer);
		if (slow) active = id;
		else timer = setTimeout(() => (active = id), STOPPED);
	}

	// A click or a tap opens the description at once, as Safari does not focus a clicked button
	function toggle(id: Feature) {
		active = id;
		selected = selected.includes(id) ? selected.filter((s) => s !== id) : [...selected, id];
	}

	// Continuing goes through the settings, which ask for a passkey next
	async function save() {
		isLoading = true;
		try {
			const config = await appConfigService.list(true);
			await appConfigService.update({ ...config, enabledFeatures: selected });
			await goto('/settings');
		} catch (e) {
			axiosErrorToast(e);
			isLoading = false;
		}
	}
</script>

<svelte:window bind:innerWidth />

<svelte:head>
	<title>{m.welcome_question()}</title>
</svelte:head>

<SignInWrapper wide>
	<AuthTile />
	<AuthHeading title={m.welcome_question()}>{m.welcome_description()}</AuthHeading>

	<div
		bind:this={grid}
		class="mb-6 grid items-start gap-6 lg:min-h-(--grid-min-h) lg:grid-cols-3 lg:gap-4"
		style:--grid-min-h="{gridMinHeight}px"
	>
		{#each sections as section, i (section.title)}
			<div
				role="group"
				aria-labelledby="{uid}-section-{i}"
				class="flex flex-col gap-2"
				style:--slot="{slots[i]}px"
			>
				<h2 id="{uid}-section-{i}" class="text-control text-fg-muted mb-1 font-semibold">
					{section.title}
				</h2>
				{#each section.rows as row (row.id)}
					{@const checked = selected.includes(row.id)}
					<button
						type="button"
						role="checkbox"
						aria-checked={checked}
						aria-describedby="{uid}-{row.id}"
						class={cn(
							'group/tile text-control focus-visible:outline-ring flex items-start gap-3 rounded-md border px-3.5 py-3 text-start font-medium transition-colors duration-(--dur-fast) focus-visible:outline-2 focus-visible:outline-offset-2 active:scale-(--press-scale)',
							checked
								? 'bg-primary-control text-primary-control-fg border-transparent'
								: 'bg-surface text-fg hover:bg-fill-row-hover'
						)}
						onclick={() => toggle(row.id)}
						onfocus={() => (active = row.id)}
						onpointermove={(event) => point(event, row.id)}
						onpointerleave={() => clearTimeout(timer)}
					>
						{#if checked}
							<CheckCircleIcon class="size-5 shrink-0" />
						{:else}
							<row.icon class="text-fg-muted size-5 shrink-0" />
						{/if}
						<span class="flex min-w-0 flex-1 flex-col">
							<span class="text-pretty wrap-anywhere">{row.title}</span>
							<!-- In the wide card the track opens to the column's shared height, and below it to the description's own height -->
							<!-- Phones cannot hover, so a ticked tile opens there instead -->
							<span
								aria-hidden="true"
								class={cn(
									'grid grid-rows-[0fr] opacity-0 transition-[grid-template-rows,opacity] duration-(--dur-base) ease-out max-lg:group-focus-visible/tile:grid-rows-[1fr] max-lg:group-focus-visible/tile:opacity-100 max-lg:group-aria-checked/tile:grid-rows-[1fr] max-lg:group-aria-checked/tile:opacity-100 lg:grid-rows-[0px]',
									active === row.id && 'lg:grid-rows-(--slot) lg:opacity-100'
								)}
							>
								<span
									data-description
									class="text-caption min-h-0 overflow-hidden pt-1 font-normal opacity-80"
								>
									{row.description}
								</span>
							</span>
						</span>
					</button>
					<span id="{uid}-{row.id}" class="sr-only">{row.description}</span>
				{/each}
			</div>
		{/each}
	</div>

	<ActionPanel>
		<div class="flex flex-col gap-3 lg:flex-row-reverse lg:items-center lg:justify-between">
			<Button class={cn(authPrimaryClass, 'lg:w-auto lg:px-8')} {isLoading} onclick={save}>
				{m.auth_continue()}
			</Button>
			<p class="text-control text-fg-muted max-lg:text-center">{m.welcome_change_later()}</p>
		</div>
	</ActionPanel>
</SignInWrapper>
