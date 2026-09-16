<script lang="ts">
	import { beforeNavigate } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import unsavedChanges from '$lib/stores/unsaved-changes-store.svelte';
	import { tick } from 'svelte';
	import { fly } from 'svelte/transition';

	let shake = $state(false);
	let tileWidth = $state<number>();

	function trackTileWidth(node: HTMLDivElement) {
		const update = () => (tileWidth = node.getBoundingClientRect().width);
		const observer = new ResizeObserver(update);
		update();
		observer.observe(node);

		return { destroy: () => observer.disconnect() };
	}

	beforeNavigate((nav) => {
		if (!unsavedChanges.hasChanges) return;

		// Cancelling a full page unload makes the browser show its own "leave site?" prompt.
		nav.cancel();
		if (nav.type !== 'leave') shake = true;
	});

	// A fresh edit supersedes any status message left over from a previous save.
	$effect(() => {
		if (unsavedChanges.hasChanges) unsavedChanges.clearStatus();
	});

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

{#if unsavedChanges.hasChanges || unsavedChanges.status}
	<div
		class="fixed inset-x-0 bottom-4 z-50 flex justify-center px-4"
		transition:fly={{ y: 20 }}
		class:animate-shake={shake}
		onanimationend={() => (shake = false)}
	>
		<div
			class="bg-popover/70 text-popover-foreground ring-foreground/5 dark:ring-foreground/10 before:pointer-events-none before:absolute before:inset-0 before:-z-1 before:rounded-[inherit] before:backdrop-blur-2xl before:backdrop-saturate-150 relative isolate overflow-hidden rounded-4xl shadow-lg ring-1 transition-[width] duration-300 ease-out"
			style:width={tileWidth ? `${tileWidth}px` : undefined}
		>
			<div class="flex w-max items-center gap-4 px-5 py-3" use:trackTileWidth>
				{#if unsavedChanges.status}
					<span
						class="text-sm font-medium {unsavedChanges.status.type === 'error'
							? 'text-red-600 dark:text-red-400'
							: 'text-green-600 dark:text-green-400'}"
					>
						{unsavedChanges.status.message}
					</span>
				{:else}
					<span class="text-sm font-medium">{m.you_have_unsaved_changes()}</span>
				{/if}
				{#if unsavedChanges.hasChanges}
					<div class="flex gap-2">
						<Button
							variant="secondary"
							size="sm"
							disabled={unsavedChanges.saving}
							onclick={() => unsavedChanges.discardAll()}
						>
							{m.discard()}
						</Button>
						<Button size="sm" isLoading={unsavedChanges.saving} onclick={saveAll}>
							{m.save()}
						</Button>
					</div>
				{/if}
			</div>
		</div>
	</div>
{/if}
