<script lang="ts">
	import CopyButton from '#lib/components/copy-button.svelte';
	import FormattedMessage from '#lib/components/formatted-message.svelte';
	import type { AnyFormattedMessage, AnyMessage } from '#lib/components/formatted-message.ts';
	import ListPagination from '#lib/components/list-pagination.svelte';
	import * as Avatar from '#lib/components/ui/avatar/index.ts';
	import { Badge } from '#lib/components/ui/badge/index.ts';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { Skeleton } from '#lib/components/ui/skeleton/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { getLocale } from '#lib/paraglide/runtime.js';
	import AuditLogService from '#lib/services/audit-log-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { AuditLog, AuditLogDateRange, AuditLogFilter } from '#lib/types/audit-log.type.ts';
	import type { ListRequestOptions, Paginated } from '#lib/types/list-request.type.ts';
	import { translateAuditLogEvent } from '#lib/utils/audit-log-translator.ts';
	import { cachedProfilePicture } from '#lib/utils/cached-image-util.ts';
	import { cn } from '#lib/utils/style.ts';
	import { LucideChevronDown, LucideMonitor, LucideSmartphone } from '@lucide/svelte';
	import { untrack } from 'svelte';
	import { SvelteSet } from 'svelte/reactivity';

	let {
		isAdmin = false,
		filters,
		dateRange = {},
		onClearFilters
	}: {
		isAdmin?: boolean;
		filters?: AuditLogFilter;
		dateRange?: AuditLogDateRange;
		onClearFilters?: () => void;
	} = $props();

	const auditLogService = new AuditLogService();

	let logs: Paginated<AuditLog> | undefined = $state();
	let isRefreshing = $state(false);
	const expanded = new SvelteSet<string>();

	const requestOptions: ListRequestOptions = {
		sort: { column: 'createdAt', direction: 'desc' },
		pagination: { page: 1, limit: 20 }
	};

	// A filter change starts again on the first page, since the old page may no longer exist
	$effect(() => {
		void filters?.userID;
		void filters?.event;
		void filters?.location;
		void filters?.clientName;
		void dateRange.from;
		void dateRange.to;
		untrack(() => {
			requestOptions.pagination!.page = 1;
			refresh();
		});
	});

	async function refresh() {
		isRefreshing = true;
		try {
			logs = isAdmin
				? await auditLogService.listAllLogs(
						{ ...requestOptions, filters: wrapFilters(filters) },
						dateRange
					)
				: await auditLogService.list(requestOptions, dateRange);
		} finally {
			isRefreshing = false;
		}
	}

	function wrapFilters(filters?: AuditLogFilter) {
		if (!filters) return undefined;
		return Object.fromEntries(
			Object.entries(filters)
				.filter(([, value]) => value)
				.map(([key, value]) => [key, [value]])
		);
	}

	const days = $derived.by(() => {
		const groups: { label: string; logs: AuditLog[] }[] = [];
		for (const log of logs?.data ?? []) {
			const label = dayLabel(new Date(log.createdAt));
			if (groups.at(-1)?.label === label) groups.at(-1)!.logs.push(log);
			else groups.push({ label, logs: [log] });
		}
		return groups;
	});

	function dayLabel(date: Date) {
		const startOfDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
		const daysAgo = Math.round((startOfDay(new Date()) - startOfDay(date)) / 86_400_000);
		if (daysAgo < 2) {
			const relative = new Intl.RelativeTimeFormat(getLocale(), { numeric: 'auto' }).format(
				-daysAgo,
				'day'
			);
			return relative.charAt(0).toUpperCase() + relative.slice(1);
		}
		return date.toLocaleDateString(getLocale(), {
			weekday: 'short',
			day: 'numeric',
			month: 'short',
			year: date.getFullYear() === new Date().getFullYear() ? undefined : 'numeric'
		});
	}

	// Events about an app or a passkey name it in the sentence
	const objectSentences: Record<string, AnyMessage | undefined> = {
		CLIENT_AUTHORIZATION: m.audit_sentence_signed_in_to,
		NEW_CLIENT_AUTHORIZATION: m.audit_sentence_allowed_app,
		DEVICE_CODE_AUTHORIZATION: m.audit_sentence_signed_in_on_device,
		NEW_DEVICE_CODE_AUTHORIZATION: m.audit_sentence_allowed_on_device,
		PASSKEY_ADDED: m.audit_sentence_added_passkey,
		PASSKEY_REMOVED: m.audit_sentence_removed_passkey
	};

	function describe(log: AuditLog): { sentence: AnyFormattedMessage | string; via?: string } {
		const signedIn = {
			message: m.audit_sentence_signed_in_to,
			inputs: { name: $appConfigStore.appName }
		};
		switch (log.event) {
			case 'SIGN_IN':
				return { sentence: signedIn, via: m.audit_via_passkey() };
			case 'TOKEN_SIGN_IN':
				return { sentence: signedIn, via: m.audit_via_login_code() };
			case 'REMOTE_SIGN_IN':
				return { sentence: signedIn, via: m.audit_via_another_device() };
			case 'ACCOUNT_CREATED':
				return { sentence: m.audit_sentence_account_created() };
		}
		const message = objectSentences[log.event];
		const name = log.data?.clientName || log.data?.passkeyName;
		if (message && name) return { sentence: { message, inputs: { name } } };
		return { sentence: translateAuditLogEvent(log.event) };
	}

	function formatLocation(log: AuditLog) {
		return [log.city, log.country].filter(Boolean).join(', ') || m.unknown();
	}

	const isPhone = (device: string) => /android|iphone|ipad|ios|mobile/i.test(device);

	const formatTime = (log: AuditLog) =>
		new Date(log.createdAt).toLocaleTimeString(getLocale(), { timeStyle: 'short' });

	const formatStamp = (log: AuditLog) =>
		new Date(log.createdAt).toLocaleString(getLocale(), {
			dateStyle: 'medium',
			timeStyle: 'medium'
		});

	function toggle(id: string) {
		if (expanded.has(id)) expanded.delete(id);
		else expanded.add(id);
	}

	const columnsClass = $derived(
		isAdmin
			? 'sm:grid-cols-[minmax(0,1fr)_minmax(0,150px)_64px_16px] lg:grid-cols-[minmax(0,180px)_minmax(0,1fr)_minmax(0,160px)_64px_16px] xl:grid-cols-[minmax(0,180px)_minmax(0,1fr)_minmax(0,200px)_minmax(0,170px)_64px_16px]'
			: 'sm:grid-cols-[minmax(0,1fr)_minmax(0,180px)_64px_16px] xl:grid-cols-[minmax(0,1fr)_minmax(0,200px)_minmax(0,170px)_64px_16px]'
	);
</script>

{#snippet sentenceText(sentence: AnyFormattedMessage | string)}
	{#if typeof sentence === 'string'}
		{sentence}
	{:else}
		<FormattedMessage message={sentence.message} inputs={sentence.inputs} />
	{/if}
{/snippet}

<div>
	<div
		class={cn(
			'transition-opacity duration-(--dur-base)',
			isRefreshing && logs && 'pointer-events-none opacity-50 delay-(--t-dim-delay)'
		)}
		aria-busy={isRefreshing}
	>
		<div
			class={cn(
				'bg-surface-sunken text-caption text-fg-muted hidden h-10 items-center gap-4 rounded-sm px-4 font-medium sm:grid',
				columnsClass
			)}
		>
			{#if isAdmin}<span class="hidden lg:block">{m.username()}</span>{/if}
			<span>{m.event()}</span>
			<span class="hidden xl:block">{m.device()}</span>
			<span>{m.ip_address()}</span>
			<span class="text-right">{m.time()}</span>
			<span></span>
		</div>

		{#if !logs}
			{#each Array.from({ length: 8 }, (_, i) => i) as i (i)}
				<div
					class="flex h-(--row-2line) items-center gap-4 border-b px-4 transition-opacity delay-(--t-skeleton-delay) starting:opacity-0"
					aria-hidden="true"
				>
					<Skeleton class="h-3.5 w-1/3" />
					<Skeleton class="ml-auto h-3.5 w-12" />
				</div>
			{/each}
		{/if}

		{#each days as day (day.label)}
			<h2 class="text-caption text-fg-muted flex h-9 items-center border-b px-4 font-medium">
				{day.label}
			</h2>
			<ul>
				{#each day.logs as log (log.id)}
					{@const { sentence, via } = describe(log)}
					{@const by =
						isAdmin && log.actorUsername && log.actorUsername !== log.username
							? log.actorUsername
							: undefined}
					{@const open = expanded.has(log.id)}
					{@const DeviceIcon = isPhone(log.device) ? LucideSmartphone : LucideMonitor}
					<li class="border-b">
						<button
							type="button"
							aria-expanded={open}
							aria-controls={`audit-detail-${log.id}`}
							class={cn(
								'hover:bg-fill-row-hover focus-visible:bg-fill-row-hover focus-visible:outline-ring grid min-h-(--row-2line) w-full grid-cols-[minmax(0,1fr)_16px] items-center gap-x-4 px-4 py-2 text-start transition-colors duration-(--dur-fast) focus-visible:outline-2 focus-visible:-outline-offset-2',
								columnsClass
							)}
							onclick={() => toggle(log.id)}
						>
							{#if isAdmin}
								<span class="hidden min-w-0 items-center gap-2 lg:flex">
									<Avatar.Root class="size-6">
										<Avatar.Image src={cachedProfilePicture.getUrl(log.userID)} alt="" />
										<Avatar.Fallback>{log.username?.[0]?.toUpperCase()}</Avatar.Fallback>
									</Avatar.Root>
									<span class="text-control truncate font-medium"
										>{log.username || m.unknown()}</span
									>
								</span>
							{/if}
							<span class="flex min-w-0 flex-col gap-0.5">
								<span class="text-body [&_b]:font-medium">
									{#if isAdmin}
										<span class="text-control font-medium lg:hidden"
											>{log.username || m.unknown()} ·</span
										>
									{/if}
									{@render sentenceText(sentence)}
									{#if by}
										<span class="text-control text-fg-muted"
											>· {m.audit_by_actor({ name: by })}</span
										>
									{/if}
								</span>
								<span class="text-caption text-fg-muted flex min-w-0 items-center gap-x-1.5">
									{#if isAdmin}
										<Badge variant="mono" class="max-sm:hidden">{log.event}</Badge>
									{/if}
									{#if log.data?.newBrowser === 'true'}
										<Badge variant="warning" title={m.new_browser_hint()}>{m.new_browser()}</Badge>
									{/if}
									{#if via}<span class="shrink-0 max-sm:hidden">{via}</span>{/if}
									<!-- Below xl the Device column folds into this line so the sentence keeps its width -->
									{#if via || isAdmin}
										<span aria-hidden="true" class="hidden sm:inline xl:hidden">·</span>
									{/if}
									<span class="hidden min-w-0 truncate sm:inline xl:hidden">{log.device}</span>
									<span class="truncate tabular-nums sm:hidden">
										{formatTime(log)} · {log.device} · {formatLocation(log)}
									</span>
								</span>
							</span>
							<span class="text-control hidden min-w-0 items-center gap-1.5 xl:flex">
								<DeviceIcon class="text-fg-muted size-4 shrink-0" />
								<span class="truncate">{log.device}</span>
							</span>
							<span class="hidden min-w-0 flex-col sm:flex">
								<span class="text-control truncate">{log.ipAddress}</span>
								<span class="text-caption text-fg-muted truncate">{formatLocation(log)}</span>
							</span>
							<time
								class="text-control hidden text-right tabular-nums sm:block"
								datetime={log.createdAt}
								title={formatStamp(log)}
							>
								{formatTime(log)}
							</time>
							<LucideChevronDown
								class={cn(
									'text-fg-muted size-3 justify-self-center transition-transform duration-(--dur-base)',
									open && 'rotate-180'
								)}
							/>
						</button>
						{#if open}
							<div
								id={`audit-detail-${log.id}`}
								class="transition-motion px-4 pt-2 pb-5 duration-(--dur-base) starting:-translate-y-1 starting:opacity-0"
							>
								<dl class="grid gap-x-6 gap-y-4 sm:grid-cols-2">
									<div class="flex min-w-0 flex-col gap-0.5">
										<dt class="text-caption text-fg-muted">{m.event()}</dt>
										<dd class="text-control flex flex-wrap items-center gap-1.5 [&_b]:font-medium">
											<span>{@render sentenceText(sentence)}</span>
											<Badge variant="mono">{log.event}</Badge>
										</dd>
									</div>
									{@render detailItem(m.time(), formatStamp(log))}
									{#if log.data?.clientName}
										{@render detailItem(m.client(), log.data.clientName)}
									{/if}
									{#if log.data?.passkeyName}
										{@render detailItem(m.passkey(), log.data.passkeyName)}
									{/if}
									{@render detailItem(m.ip_address(), log.ipAddress, m.copy_ip_address())}
									{@render detailItem(m.approximate_location(), formatLocation(log))}
									{@render detailItem(
										m.device(),
										log.data?.newBrowser === 'true'
											? `${log.device} · ${m.new_browser_hint()}`
											: log.device
									)}
									{@render detailItem(m.event_id(), log.id, m.copy_event_id())}
								</dl>
								<details class="group/raw mt-4">
									<summary
										class="text-control text-fg-muted hover:text-fg inline-flex cursor-pointer list-none items-center gap-1 rounded-xs"
									>
										<LucideChevronDown
											class="size-3 -rotate-90 transition-transform duration-(--dur-base) group-open/raw:rotate-0"
										/>
										{m.raw_data()}
									</summary>
									<div class="bg-surface-sunken relative mt-2 rounded-md border p-4">
										<pre
											class="text-mono-s font-mono wrap-anywhere whitespace-pre-wrap max-sm:pt-10 sm:pr-28">{JSON.stringify(
												log,
												null,
												2
											)}</pre>
										<CopyButton
											labeled
											value={JSON.stringify(log, null, 2)}
											label={m.copy_json()}
											class="absolute top-3 right-3"
										/>
									</div>
								</details>
							</div>
						{/if}
					</li>
				{/each}
			</ul>
		{/each}

		{#if logs?.pagination.totalItems === 0}
			<p
				class="text-control text-fg-muted flex min-h-(--row-2line) flex-wrap items-center gap-x-3 border-b px-4"
			>
				{m.no_events_match_filters()}
				{#if onClearFilters}
					<Button variant="link" onclick={onClearFilters}>{m.clear_filters()}</Button>
				{/if}
			</p>
		{/if}
	</div>

	<ListPagination
		pagination={logs?.pagination.totalItems ? logs.pagination : undefined}
		onPageChange={(page) => {
			requestOptions.pagination!.page = page;
			refresh();
		}}
		onPageSizeChange={(limit) => {
			requestOptions.pagination = { page: 1, limit };
			refresh();
		}}
	/>
</div>

{#snippet detailItem(label: string, value: string, copyLabel?: string)}
	<div class="flex min-w-0 flex-col gap-0.5">
		<dt class="text-caption text-fg-muted">{label}</dt>
		<dd class="text-control flex min-w-0 items-center gap-1.5 wrap-anywhere">
			{#if copyLabel}
				<span class="font-mono">{value}</span>
				<CopyButton {value} label={copyLabel} />
			{:else}
				{value}
			{/if}
		</dd>
	</div>
{/snippet}
