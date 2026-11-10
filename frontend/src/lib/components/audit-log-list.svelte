<script lang="ts">
	import CopyButton from '#lib/components/copy-button.svelte';
	import ListPagination from '#lib/components/list-pagination.svelte';
	import OidcClientAvatar from '#lib/components/oidc-client-avatar.svelte';
	import AdvancedTableColumnSelection from '#lib/components/table/advanced-table-column-selection.svelte';
	import * as Avatar from '@pocket-id/ui/avatar';
	import { Badge } from '@pocket-id/ui/badge';
	import { Button } from '@pocket-id/ui/button';
	import { Skeleton } from '@pocket-id/ui/skeleton';
	import { m } from '#lib/paraglide/messages.js';
	import { getLocale } from '#lib/paraglide/runtime.js';
	import AuditLogService from '#lib/services/audit-log-service.ts';
	import type { AdvancedTableColumn } from '#lib/types/advanced-table.type.ts';
	import type { AuditLog, AuditLogDateRange, AuditLogFilter } from '#lib/types/audit-log.type.ts';
	import type { ListRequestOptions, Paginated } from '#lib/types/list-request.type.ts';
	import { translateAuditLogEvent } from '#lib/utils/audit-log-translator.ts';
	import { cachedProfilePicture } from '#lib/utils/cached-asset-util.ts';
	import { cn } from '@pocket-id/ui/utils';
	import ChevronDownIcon from '~icons/solar/alt-arrow-down-linear';
	import CodeSquareIcon from '~icons/solar/code-square-bold-duotone';
	import NotebookIcon from '~icons/solar/notebook-bold-duotone';
	import SettingsIcon from '~icons/solar/settings-bold-duotone';
	import { PersistedState } from 'runed';
	import { untrack, type Component } from 'svelte';
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

	const signInMethods: Record<string, (() => string) | undefined> = {
		SIGN_IN: m.audit_via_passkey,
		TOKEN_SIGN_IN: m.audit_via_login_code,
		REMOTE_SIGN_IN: m.audit_via_another_device
	};

	// Revoking every session at once names no single device
	function signedOutDevice(log: AuditLog) {
		if (!log.data?.sessionCount) return log.data?.device;
		return log.data.exceptCurrent === 'true' ? m.audit_all_other_devices() : m.audit_all_devices();
	}

	const actorIcons: Partial<Record<AuditLog['actorType'], Component>> = {
		ldap: NotebookIcon,
		cli: CodeSquareIcon,
		system: SettingsIcon
	};

	// A sign-in names its method, and a change made with an API key or through an app names it, while a session or a client acting as itself needs no mention
	function credentialVia(log: AuditLog) {
		const method = signInMethods[log.event];
		if (method) return method();
		const name = log.data?.viaName;
		if (!name) return undefined;
		switch (log.data?.via) {
			case 'api_key':
				return m.audit_via_api_key({ name });
			case 'mcp':
				return m.audit_via_mcp({ name });
			case 'oauth':
				return m.audit_via_app({ name });
		}
	}

	function actorName(log: AuditLog) {
		switch (log.actorType) {
			case 'user':
			case 'client':
				return log.actorName || m.unknown();
			case 'ldap':
				return m.audit_actor_ldap();
			case 'cli':
				return m.audit_actor_cli();
			case 'system':
				return m.system();
		}
	}

	function formatLocation(log: AuditLog) {
		if (!log.ipAddress) return '';
		return [log.city, log.country].filter(Boolean).join(', ') || m.unknown();
	}

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

	type Column = AdvancedTableColumn<AuditLog> & { key: string; width: string };

	// The event names the row, so it always shows first
	const adminColumns: Column[] = [
		{ key: 'event', label: m.event(), width: 'minmax(0,1fr)' },
		{ key: 'actor', label: m.performed_by(), width: 'minmax(0,1fr)' },
		{ key: 'resource', label: m.audit_resource(), width: 'minmax(0,1fr)' },
		{ key: 'time', label: m.time(), width: '72px' },
		{ key: 'location', label: m.location_and_device(), width: 'minmax(0,1fr)', hidden: true },
		{ key: 'ipAddress', label: m.ip_address(), width: 'minmax(0,140px)', hidden: true }
	];

	// Users get a fixed set that shows whether each entry was really them, so who acted moves under the event when it wasn't them
	const userColumns = ['event', 'resource', 'location', 'time'].map((key) =>
		adminColumns.find((c) => c.key === key)!
	);

	const preferences = new PersistedState<{ visibleColumns: string[]; columnOrder?: string[] }>(
		'audit-log-columns',
		{ visibleColumns: adminColumns.filter((c) => !c.hidden).map((c) => c.key) }
	);

	// Columns the saved order doesn't know yet, such as ones added in an update, keep their place after the ordered ones
	const orderedColumns = $derived.by(() => {
		const order = preferences.current.columnOrder ?? [];
		const rank = (column: Column, index: number) => {
			const position = order.indexOf(column.key);
			return position === -1 ? order.length + index : position;
		};
		const [event, ...rest] = adminColumns;
		return [
			event,
			...rest
				.map((column, index) => ({ column, rank: rank(column, index) }))
				.sort((a, b) => a.rank - b.rank)
				.map(({ column }) => column)
		];
	});

	const visibleColumns = $derived(
		isAdmin
			? orderedColumns.filter(
					(c, i) => i === 0 || preferences.current.visibleColumns.includes(c.key)
				)
			: userColumns
	);

	const gridColumns = $derived([...visibleColumns.map((c) => c.width), '16px'].join(' '));
</script>

<div>
	<div
		class={cn(
			'transition-opacity duration-(--dur-base)',
			isRefreshing && logs && 'pointer-events-none opacity-50 delay-(--t-dim-delay)'
		)}
		aria-busy={isRefreshing}
	>
		<div
			class="bg-surface-sunken text-caption text-fg-muted mb-1 hidden h-10 grid-cols-(--columns) items-center gap-4 rounded-sm px-4 font-medium sm:grid"
			style:--columns={gridColumns}
		>
			{#each visibleColumns as column (column.key)}
				<span class="truncate">{column.label}</span>
			{/each}
			<span class="flex justify-center">
				{#if isAdmin}
					<AdvancedTableColumnSelection
						columns={orderedColumns}
						bind:selectedColumns={preferences.current.visibleColumns}
						bind:columnOrder={preferences.current.columnOrder}
					/>
				{/if}
			</span>
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
			<h2
				class="text-caption text-fg-muted flex h-9 items-center border-b px-4 font-medium transition-colors duration-(--dur-fast) has-[+ul>li:first-child>button:is(:hover,:focus-visible)]:border-transparent"
			>
				{day.label}
			</h2>
			<ul>
				{#each day.logs as log (log.id)}
					<!-- Deleted users keep their username but lose their ID -->
					{@const selfAction =
						log.actorType === 'user' &&
						(log.userID ? log.actorID === log.userID : log.actorName === log.username)}
					<!-- The user is the resource only when someone else acted on them, while users acting on themselves show what they changed -->
					<!-- Users' own activity is all about them, so it never names them -->
					{@const subject =
						isAdmin && !selfAction && (log.userID || log.username)
							? log.username || m.unknown()
							: undefined}
					{@const actor = actorName(log)}
					{@const via = credentialVia(log)}
					{@const performedBy = [actor, via].filter(Boolean).join(' ')}
					{@const actedBy = selfAction
						? via
						: actor && [m.audit_by({ name: actor }), via].filter(Boolean).join(' ')}
					{@const objectName =
						log.data?.groupName ||
						log.data?.apiKeyName ||
						(subject
							? undefined
							: log.data?.passkeyName || log.data?.email || signedOutDevice(log))}
					<!-- A client acting on itself, such as rotating its own secret, is already named as the actor -->
					{@const client =
						log.actorType === 'client' && log.actorID === log.data?.clientID
							? undefined
							: log.data?.clientName}
					{@const resources = [subject, client, objectName].filter(Boolean)}
					{@const open = expanded.has(log.id)}
					<!-- Like table rows, a filled row hides the dividers around it, but an open row keeps the one below its details -->
					<li
						class="border-b transition-colors duration-(--dur-fast) has-[+li>button:is(:hover,:focus-visible)]:border-transparent has-[>button[aria-expanded=false]:is(:hover,:focus-visible)]:border-transparent"
					>
						<button
							type="button"
							aria-expanded={open}
							aria-controls={`audit-detail-${log.id}`}
							class="hover:bg-fill-row-hover focus-visible:bg-fill-row-hover focus-visible:outline-ring text-control grid min-h-(--row-2line) w-full grid-cols-[minmax(0,1fr)_16px] items-center gap-x-4 rounded-sm px-4 py-2 text-start transition-colors duration-(--dur-fast) focus-visible:outline-2 focus-visible:-outline-offset-2 sm:grid-cols-(--columns)"
							style:--columns={gridColumns}
							onclick={() => toggle(log.id)}
						>
							{#each visibleColumns as column (column.key)}
								{#if column.key === 'event'}
									<span class="flex min-w-0 flex-col gap-0.5">
										<span class="flex min-w-0 items-center gap-2">
											<span class="truncate font-medium">{translateAuditLogEvent(log.event)}</span>
											{#if log.data?.newBrowser === 'true'}
												<Badge variant="warning" class="shrink-0" title={m.new_browser_hint()}>
													{m.new_browser()}
												</Badge>
											{/if}
										</span>
										{#if !isAdmin && actedBy}
											<span class="text-caption text-fg-muted truncate max-sm:hidden"
												>{actedBy}</span
											>
										{/if}
										<span class="text-caption text-fg-muted truncate tabular-nums sm:hidden">
											{[formatTime(log), isAdmin ? performedBy : actedBy, resources.join(', ')]
												.filter(Boolean)
												.join(' · ')}
										</span>
									</span>
								{:else if column.key === 'actor'}
									{@const ActorIcon = actorIcons[log.actorType]}
									<span class="hidden min-w-0 items-center gap-2 sm:flex">
										{#if log.actorType === 'user'}
											{@render userAvatar(log.actorID, log.actorName)}
										{:else if log.actorType === 'client'}
											<OidcClientAvatar id={log.actorID} name={log.actorName} hasLogo size={20} />
										{:else if ActorIcon}
											<span
												class="bg-fill-muted text-fg-muted inset-ring-border inline-grid size-5 shrink-0 place-items-center rounded-[5px] inset-ring"
											>
												<ActorIcon class="size-3.5" />
											</span>
										{/if}
										<span class="flex min-w-0 flex-col">
											<span class="truncate">{actor}</span>
											{#if via}<span class="text-caption text-fg-muted truncate">{via}</span>{/if}
										</span>
									</span>
								{:else if column.key === 'resource'}
									<!-- Entries recorded before the client ID was stored show the monogram -->
									<span class="hidden min-w-0 items-center gap-2 sm:flex">
										{#if subject}
											{@render userAvatar(log.userID, log.username ?? subject)}
										{:else if client}
											<OidcClientAvatar
												id={log.data.clientID ?? ''}
												name={client}
												hasLogo={Boolean(log.data.clientID)}
												size={20}
											/>
										{/if}
										<span class="flex min-w-0 flex-col">
											<span class="truncate">{resources[0]}</span>
											{#if resources.length > 1}
												<span class="text-caption text-fg-muted truncate">
													{resources.slice(1).join(', ')}
												</span>
											{/if}
										</span>
									</span>
								{:else if column.key === 'time'}
									<time
										class="hidden whitespace-nowrap tabular-nums sm:block"
										datetime={log.createdAt}
										title={formatStamp(log)}
									>
										{formatTime(log)}
									</time>
								{:else if column.key === 'location'}
									<!-- Entries without an IP address show the device alone -->
									{@const [place, device] = [formatLocation(log), log.device].filter(Boolean)}
									<span class="hidden min-w-0 flex-col sm:flex">
										<span class="truncate">{place}</span>
										{#if device}<span class="text-caption text-fg-muted truncate">{device}</span
											>{/if}
									</span>
								{:else}
									<span class="hidden truncate tabular-nums sm:block">{log.ipAddress}</span>
								{/if}
							{/each}
							<ChevronDownIcon
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
										<dd class="text-control flex flex-wrap items-center gap-1.5">
											<span>{translateAuditLogEvent(log.event)}</span>
											<Badge variant="mono">{log.event}</Badge>
										</dd>
									</div>
									{@render detailItem(m.time(), formatStamp(log))}
									{#if performedBy}
										{@render detailItem(m.performed_by(), performedBy)}
									{/if}
									{#if log.data?.clientName}
										{@render detailItem(m.client(), log.data.clientName)}
									{/if}
									{#if log.data?.groupName}
										{@render detailItem(m.group(), log.data.groupName)}
									{/if}
									{#if log.data?.passkeyName}
										{@render detailItem(m.passkey(), log.data.passkeyName)}
									{/if}
									{#if log.data?.email}
										{@render detailItem(m.email(), log.data.email)}
									{/if}
									{#if log.data?.previousEmail}
										{@render detailItem(m.audit_previous_email(), log.data.previousEmail)}
									{/if}
									{#if log.event === 'SESSION_REVOKED'}
										{@render detailItem(
											m.audit_signed_out_device(),
											signedOutDevice(log) ?? m.unknown()
										)}
									{/if}
									{#if log.data?.changes}
										{@render detailItem(m.audit_changed_settings(), log.data.changes)}
									{/if}
									{#if log.ipAddress}
										{@render detailItem(m.ip_address(), log.ipAddress, m.copy_ip_address())}
										{@render detailItem(m.approximate_location(), formatLocation(log))}
									{/if}
									{#if log.device}
										{@render detailItem(
											m.device(),
											log.data?.newBrowser === 'true'
												? `${log.device} · ${m.new_browser_hint()}`
												: log.device
										)}
									{/if}
									{@render detailItem(m.event_id(), log.id, m.copy_event_id())}
								</dl>
								<details class="group/raw mt-4">
									<summary
										class="text-control text-fg-muted hover:text-fg inline-flex cursor-pointer list-none items-center gap-1 rounded-xs"
									>
										<ChevronDownIcon
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

{#snippet userAvatar(id: string, name: string)}
	<Avatar.Root class="size-5 shrink-0">
		<Avatar.Image src={cachedProfilePicture.getUrl(id)} alt="" />
		<Avatar.Fallback>{name[0]?.toUpperCase()}</Avatar.Fallback>
	</Avatar.Root>
{/snippet}

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
