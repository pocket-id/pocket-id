<script lang="ts">
	import CopyButton from '#lib/components/copy-button.svelte';
	import { Button } from '@pocket-id/ui/button';
	import * as Sheet from '@pocket-id/ui/sheet';
	import { m } from '#lib/paraglide/messages.js';
	import { getLocale } from '#lib/paraglide/runtime.js';
	import type { Session } from '#lib/types/session.type.ts';
	import {
		sessionDeviceName,
		sessionLocation,
		sessionSignInMethod
	} from '#lib/utils/session-util.ts';

	let {
		open = $bindable(),
		session,
		onRevoke
	}: {
		open: boolean;
		session: Session | null;
		onRevoke: (session: Session) => void;
	} = $props();

	const formatDate = (date: string) =>
		new Date(date).toLocaleString(getLocale(), { dateStyle: 'medium', timeStyle: 'short' });
</script>

<Sheet.Root bind:open>
	<Sheet.Content title={session ? sessionDeviceName(session) : ''}>
		{#if session}
			<dl class="grid gap-x-6 gap-y-4 sm:grid-cols-2">
				{@render detailItem(m.signed_in(), formatDate(session.createdAt))}
				{@render detailItem(m.sign_in_method(), sessionSignInMethod(session))}
				{@render detailItem(m.last_active(), formatDate(session.lastActiveAt))}
				{@render detailItem(m.expires(), formatDate(session.expiresAt))}
				{@render detailItem(m.approximate_location(), sessionLocation(session))}
				<div class="flex min-w-0 flex-col gap-0.5">
					<dt class="text-caption text-fg-muted">{m.ip_address()}</dt>
					<dd class="text-control flex min-w-0 items-center gap-1.5">
						<span class="font-mono">{session.ipAddress || m.unknown()}</span>
						{#if session.ipAddress}
							<CopyButton value={session.ipAddress} label={m.copy_ip_address()} />
						{/if}
					</dd>
				</div>
				<div class="flex min-w-0 flex-col gap-0.5 sm:col-span-2">
					<dt class="text-caption text-fg-muted">{m.user_agent()}</dt>
					<dd class="text-mono-s font-mono wrap-anywhere">{session.userAgent || m.unknown()}</dd>
				</div>
			</dl>
		{/if}
		{#snippet footer()}
			<Button variant="ghost" onclick={() => (open = false)}>{m.close()}</Button>
			{#if session && !session.current}
				<Button variant="destructive" onclick={() => onRevoke(session)}>{m.sign_out()}</Button>
			{/if}
		{/snippet}
	</Sheet.Content>
</Sheet.Root>

{#snippet detailItem(label: string, value: string)}
	<div class="flex min-w-0 flex-col gap-0.5">
		<dt class="text-caption text-fg-muted">{label}</dt>
		<dd class="text-control wrap-anywhere">{value}</dd>
	</div>
{/snippet}
