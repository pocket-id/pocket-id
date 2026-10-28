<script lang="ts">
	import { Button } from '#lib/components/ui/button/index.ts';
	import { Input } from '#lib/components/ui/input/index.ts';
	import * as Popover from '#lib/components/ui/popover/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { getLocale } from '#lib/paraglide/runtime.js';
	import type { AuditLogDateRange } from '#lib/types/audit-log.type.ts';
	import { preventDefault } from '#lib/utils/event-util.ts';
	import { Check, ChevronDown } from '@lucide/svelte';

	let { range = $bindable({}) }: { range: AuditLogDateRange } = $props();

	const DAY = 86_400_000;
	const presets = [
		{ id: '24h', label: m.last_24_hours(), ms: DAY },
		{ id: '7d', label: m.last_n_days({ count: 7 }), ms: 7 * DAY },
		{ id: '30d', label: m.last_n_days({ count: 30 }), ms: 30 * DAY },
		{ id: '90d', label: m.last_n_days({ count: 90 }), ms: 90 * DAY }
	];

	let open = $state(false);
	let selected = $state('any');
	let label = $state<string>(m.any());
	let from = $state('');
	let to = $state('');
	const invalid = $derived(!!from && !!to && to < from);

	// Times are shown in the browser's zone, so the trigger names it
	const zone =
		new Intl.DateTimeFormat(getLocale(), { timeZoneName: 'shortOffset' })
			.formatToParts(new Date())
			.find((p) => p.type === 'timeZoneName')?.value ?? '';

	function choose(preset: (typeof presets)[number]) {
		selected = preset.id;
		label = preset.label;
		range = preset.ms ? { from: new Date(Date.now() - preset.ms).toISOString() } : {};
		open = false;
	}

	function day(value: string) {
		const date = new Date(`${value}T00:00`);
		return date.toLocaleDateString(getLocale(), {
			day: 'numeric',
			month: 'short',
			year: date.getFullYear() === new Date().getFullYear() ? undefined : 'numeric'
		});
	}

	// A custom range covers whole local days, so the end is the start of the day after the To date
	function applyCustom() {
		if (invalid || (!from && !to)) return;
		selected = 'custom';
		label =
			from && from === to ? day(from) : `${from ? day(from) : ''} – ${to ? day(to) : ''}`.trim();
		const startOf = (value: string, addDays = 0) =>
			new Date(new Date(`${value}T00:00`).getTime() + addDays * DAY).toISOString();
		range = { from: from ? startOf(from) : undefined, to: to ? startOf(to, 1) : undefined };
		open = false;
	}
</script>

<Popover.Root bind:open>
	<Popover.Trigger
		aria-label={m.facet_filter_label({ name: m.date(), value: label })}
		class="hover:bg-fill-hover aria-expanded:bg-fill-selected focus-visible:outline-ring text-control flex h-7 shrink-0 items-center gap-1 rounded-xs pr-1.5 pl-2 whitespace-nowrap transition-colors duration-(--dur-fast) focus-visible:outline-2 focus-visible:outline-offset-2"
	>
		<span class="text-fg-muted">{m.date()}:</span>
		<span class={selected === 'any' ? 'text-fg-muted' : 'text-fg font-medium'}>{label}</span>
		<span class="text-fg-muted">({zone})</span>
		<ChevronDown class="text-fg-muted size-3" />
	</Popover.Trigger>
	<Popover.Content align="start" class="w-72 gap-0 p-1">
		{#each [{ id: 'any', label: m.any(), ms: 0 }, ...presets] as preset (preset.id)}
			<button
				type="button"
				aria-pressed={selected === preset.id}
				class="hover:bg-fg/6 focus-visible:outline-ring text-control flex h-9 w-full items-center gap-2 rounded-sm px-2.5 text-left focus-visible:outline-2 focus-visible:-outline-offset-2"
				onclick={() => choose(preset)}
			>
				<span class="flex size-4 items-center justify-center">
					{#if selected === preset.id}<Check class="size-4" />{/if}
				</span>
				{preset.label}
			</button>
		{/each}
		<form
			class="mt-1 flex flex-col gap-3 border-t px-2.5 pt-3 pb-2"
			onsubmit={preventDefault(applyCustom)}
		>
			<p class="text-caption text-fg-muted font-medium">{m.custom_range()}</p>
			<div class="flex flex-col gap-2">
				<label class="text-caption text-fg-muted flex flex-col gap-1">
					{m.date_from()}
					<Input type="date" bind:value={from} aria-invalid={invalid} />
				</label>
				<label class="text-caption text-fg-muted flex flex-col gap-1">
					{m.date_to()}
					<Input type="date" bind:value={to} aria-invalid={invalid} />
				</label>
			</div>
			{#if invalid}<p class="text-caption text-danger-fg">{m.date_range_order()}</p>{/if}
			<Button type="submit" size="sm" variant="secondary" disabled={invalid || (!from && !to)}>
				{m.apply()}
			</Button>
		</form>
	</Popover.Content>
</Popover.Root>
