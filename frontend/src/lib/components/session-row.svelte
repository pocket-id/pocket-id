<script lang="ts">
	import { Button } from '@pocket-id/ui/button';
	import { m } from '#lib/paraglide/messages.js';
	import { getLocale } from '#lib/paraglide/runtime.js';
	import type { Session } from '#lib/types/session.type.ts';
	import { sessionDeviceName, sessionLocation } from '#lib/utils/session-util.ts';
	import { LucideMonitor, LucideSmartphone, LucideTablet } from '@lucide/svelte';

	let {
		session,
		onSelect,
		onRevoke
	}: {
		session: Session;
		onSelect: () => void;
		onRevoke: () => void;
	} = $props();

	const DeviceIcon = $derived(
		{ desktop: LucideMonitor, mobile: LucideSmartphone, tablet: LucideTablet }[session.deviceType]
	);

	function lastActive() {
		const seconds = (new Date(session.lastActiveAt).getTime() - Date.now()) / 1000;
		const format = new Intl.RelativeTimeFormat(getLocale(), { numeric: 'auto' });
		const units: [Intl.RelativeTimeFormatUnit, number][] = [
			['day', 86400],
			['hour', 3600],
			['minute', 60]
		];
		const [unit, size] = units.find(([, size]) => Math.abs(seconds) >= size) ?? ['minute', 60];
		return format.format(Math.round(seconds / size), unit);
	}
</script>

<div class="flex min-h-(--row-2line) items-center gap-3 px-4 py-2.5">
	<button
		type="button"
		class="focus-visible:outline-ring flex min-w-0 flex-1 items-center gap-3 rounded-xs text-start focus-visible:outline-2 focus-visible:outline-offset-2"
		onclick={onSelect}
	>
		<span class="bg-fill-muted text-fg-muted grid size-8 shrink-0 place-items-center rounded-[8px]">
			<DeviceIcon class="size-4" />
		</span>
		<span class="flex min-w-0 flex-1 flex-col gap-0.5">
			<span class="text-heading truncate">{sessionDeviceName(session)}</span>
			<span class="text-caption text-fg-muted truncate">
				{#if session.current}
					<span class="bg-success-solid me-0.5 inline-block size-1.5 rounded-full align-middle"
					></span>
					{m.this_device()} · {sessionLocation(session)}
				{:else}
					{sessionLocation(session)} · {m.session_last_active({ time: lastActive() })}
				{/if}
			</span>
		</span>
	</button>
	{#if !session.current}
		<Button variant="secondary" size="sm" onclick={onRevoke}>{m.sign_out()}</Button>
	{/if}
</div>
