<script lang="ts">
	import ActionPanel from '#lib/components/auth/action-panel.svelte';
	import AuthHeading from '#lib/components/auth/auth-heading.svelte';
	import AuthTile from '#lib/components/auth/auth-tile.svelte';
	import { authPrimaryClass } from '#lib/components/auth/auth-util.ts';
	import SignInWrapper from '#lib/components/login-wrapper.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import AppConfigService from '#lib/services/app-config-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { Feature } from '#lib/types/feature.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { featureSections } from '#lib/utils/feature-util.ts';
	import { cn } from '#lib/utils/style.ts';
	import { goto } from '$app/navigation';
	import { LucideCheck, LucidePlus } from '@lucide/svelte';

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
	// Titles that wrap in longer languages change the heights once the web fonts arrive
	let fontsLoaded = $state(false);
	document.fonts.ready.then(() => (fontsLoaded = true));

	// A tile grows to show its description, so the grid keeps room for the tallest column with its tallest description and the card never changes size
	$effect(() => {
		void innerWidth;
		void fontsLoaded;
		if (!grid) return;
		const columns = [...grid.children] as HTMLElement[];
		gridMinHeight = Math.max(
			0,
			...columns.map(
				(column) =>
					column.offsetHeight +
					Math.max(
						0,
						...[...column.querySelectorAll<HTMLElement>('[data-description]')].map(
							(d) => d.scrollHeight
						)
					)
			)
		);
	});

	function toggle(id: Feature) {
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

<SignInWrapper wide columns={false}>
	<AuthTile />
	<AuthHeading title={m.welcome_question()}>
		{m.welcome_description()}
	</AuthHeading>

	<div
		bind:this={grid}
		class="mb-6 grid items-start gap-6 lg:min-h-(--grid-min-h) lg:grid-cols-3 lg:gap-4"
		style:--grid-min-h="{gridMinHeight}px"
	>
		{#each sections as section, i (section.title)}
			<div role="group" aria-labelledby="{uid}-section-{i}" class="flex flex-col gap-2">
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
								? 'bg-primary-control text-primary-control-foreground border-transparent'
								: 'bg-surface text-fg hover:bg-fill-row-hover'
						)}
						onclick={() => toggle(row.id)}
					>
						<row.icon class={cn('mt-0.5 size-4 shrink-0', !checked && 'text-fg-muted')} />
						<span class="flex min-w-0 flex-1 flex-col">
							<span class="text-pretty wrap-anywhere">{row.title}</span>
							<!-- The row track grows from nothing, so the description opens at its own height -->
							<!-- Phones cannot hover, so a ticked tile opens there instead -->
							<span
								aria-hidden="true"
								class="grid grid-rows-[0fr] opacity-0 transition-[grid-template-rows,opacity] duration-(--dur-base) ease-out group-focus-visible/tile:grid-rows-[1fr] group-focus-visible/tile:opacity-100 max-lg:group-aria-checked/tile:grid-rows-[1fr] max-lg:group-aria-checked/tile:opacity-100 lg:group-hover/tile:grid-rows-[1fr] lg:group-hover/tile:opacity-100"
							>
								<span
									data-description
									class="text-caption min-h-0 overflow-hidden pt-1 font-normal opacity-80"
								>
									{row.description}
								</span>
							</span>
						</span>
						{#if checked}
							<LucideCheck class="mt-0.5 size-4 shrink-0" />
						{:else}
							<LucidePlus class="text-fg-muted mt-0.5 size-4 shrink-0" />
						{/if}
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
