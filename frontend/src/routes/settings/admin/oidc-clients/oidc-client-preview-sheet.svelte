<script lang="ts">
	import CopyButton from '#lib/components/copy-button.svelte';
	import MultiSelect from '#lib/components/form/multi-select.svelte';
	import SearchableSelect from '#lib/components/form/searchable-select.svelte';
	import { Badge } from '#lib/components/ui/badge/index.ts';
	import * as Field from '#lib/components/ui/field/index.ts';
	import * as Sheet from '#lib/components/ui/sheet/index.ts';
	import { Skeleton } from '#lib/components/ui/skeleton/index.ts';
	import * as Tabs from '#lib/components/ui/tabs/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { getLocale } from '#lib/paraglide/runtime.js';
	import OidcService from '#lib/services/oidc-service.ts';
	import UserService from '#lib/services/user-service.ts';
	import type { OidcClient } from '#lib/types/oidc.type.ts';
	import type { User } from '#lib/types/user.type.ts';
	import { debounced } from '#lib/utils/debounce-util.ts';
	import { getAxiosErrorMessage } from '#lib/utils/error-util.ts';
	import { cn } from '#lib/utils/style.ts';
	import { LucideBraces, LucideList, LucideLock } from '@lucide/svelte';

	type Claims = Record<string, unknown>;

	let {
		open = $bindable(),
		client
	}: {
		open: boolean;
		client: OidcClient;
	} = $props();

	const oidcService = new OidcService();
	const userService = new UserService();

	// Claims holding a Unix timestamp, which are shown with a readable date next to them
	const TIMESTAMP_CLAIMS = new Set(['exp', 'iat', 'nbf', 'auth_time', 'updated_at']);

	let previewData = $state<{ idToken?: Claims; accessToken?: Claims; userInfo?: Claims } | null>(
		null
	);
	let isLoading = $state(false);
	let isUserSearchLoading = $state(false);
	let user: User | null = $state(null);
	let users: User[] = $state([]);
	// openid is always requested, so only the other scopes can be picked
	let scopes: string[] = $state(['email', 'profile']);
	let errorMessage: string | null = $state(null);
	let view = $state<'claims' | 'json'>('claims');
	let activeTab = $state<'idToken' | 'accessToken' | 'userInfo'>('idToken');

	// Each request gets a number so that responses arriving out of order don't overwrite newer ones
	let latestRequest = 0;

	const activeData = $derived(previewData?.[activeTab] ?? {});
	const activeJson = $derived(JSON.stringify(activeData, null, 2));

	async function loadPreviewData(userId: string, scope: string) {
		const request = ++latestRequest;
		isLoading = true;
		errorMessage = null;

		try {
			const data = await oidcService.getClientPreview(client.id, userId, scope);
			if (request === latestRequest) previewData = data;
		} catch (e) {
			if (request !== latestRequest) return;
			errorMessage = getAxiosErrorMessage(e);
			previewData = null;
		} finally {
			if (request === latestRequest) isLoading = false;
		}
	}

	async function loadUsers(search?: string) {
		users = (await userService.list({ search, pagination: { limit: 10, page: 1 } })).data;
		user ??= users[0] ?? null;
	}

	const onUserSearch = debounced(
		async (search: string) => await loadUsers(search),
		300,
		(loading) => (isUserSearchLoading = loading)
	);

	// Timestamps arrive either as Unix seconds or as ISO strings depending on the claim, so both are accepted
	function formatTimestamp(value: unknown) {
		const date =
			typeof value === 'number'
				? new Date(value * 1000)
				: typeof value === 'string'
					? new Date(value)
					: null;
		return date && !isNaN(date.getTime())
			? date.toLocaleString(getLocale(), { dateStyle: 'medium', timeStyle: 'short' })
			: null;
	}

	// The user list is only needed once the sheet is opened
	$effect(() => {
		if (open && users.length === 0) {
			loadUsers();
		}
	});

	// Reload the preview whenever the selected user or the scopes change
	$effect(() => {
		if (open && user) {
			loadPreviewData(user.id, ['openid', ...scopes].join(' '));
		}
	});
</script>

{#snippet claimValue(key: string, value: unknown)}
	{#if Array.isArray(value) && value.every((item) => typeof item !== 'object')}
		<span class="flex flex-wrap gap-1.5">
			{#each value as item, i (i)}
				<Badge variant="mono" class="h-auto max-w-full break-all whitespace-normal"
					>{String(item)}</Badge
				>
			{:else}
				<span class="text-fg-muted font-mono">[]</span>
			{/each}
		</span>
	{:else if value !== null && typeof value === 'object'}
		<pre class="text-mono-s font-mono break-all whitespace-pre-wrap">{JSON.stringify(
				value,
				null,
				2
			)}</pre>
	{:else if value === '' || value === null}
		<span class="text-fg-muted font-mono">{value === null ? 'null' : '""'}</span>
	{:else}
		{@const formattedDate = TIMESTAMP_CLAIMS.has(key) ? formatTimestamp(value) : null}
		<span class="flex min-w-0 items-center gap-1">
			<span class="min-w-0 wrap-anywhere">
				<span class={cn(typeof value !== 'string' || formattedDate ? 'font-mono' : '')}>
					{String(value)}
				</span>
				{#if formattedDate}
					<span class="text-fg-muted">· {formattedDate}</span>
				{/if}
			</span>
			<CopyButton value={String(value)} label={m.copy_label({ label: key })} />
		</span>
	{/if}
{/snippet}

<Sheet.Root bind:open>
	<Sheet.Content title={m.data_preview()}>
		<div class="grid gap-4 sm:grid-cols-2">
			<Field.Field>
				<Field.Label>{m.person()}</Field.Label>
				<SearchableSelect
					selectText={m.select_user()}
					isLoading={isUserSearchLoading}
					items={users.map((user) => ({ value: user.id, label: user.username }))}
					value={user?.id || ''}
					oninput={(e) => onUserSearch(e.currentTarget.value)}
					onSelect={(value) => (user = users.find((u) => u.id === value) || null)}
				/>
			</Field.Field>
			<Field.Field>
				<Field.Label>{m.scopes()}</Field.Label>
				<div class="flex items-center gap-2">
					<Badge variant="mono" title={m.always_included()}>
						<LucideLock />
						openid
					</Badge>
					<MultiSelect
						class="min-w-0 flex-1"
						placeholder={m.only_openid()}
						items={['email', 'profile', 'groups'].map((scope) => ({ value: scope, label: scope }))}
						bind:selectedItems={scopes}
					/>
				</div>
			</Field.Field>
		</div>

		<div class="flex min-h-0 flex-col gap-3">
			<Tabs.Root bind:value={activeTab}>
				<Tabs.List variant="line" aria-label={m.token()}>
					<Tabs.Trigger value="idToken">{m.id_token()}</Tabs.Trigger>
					<Tabs.Trigger value="accessToken">{m.access_token()}</Tabs.Trigger>
					<Tabs.Trigger value="userInfo">{m.userinfo()}</Tabs.Trigger>
				</Tabs.List>
			</Tabs.Root>
			<div class="flex flex-wrap items-center justify-between gap-3">
				<Tabs.Root bind:value={view}>
					<Tabs.List aria-label={m.view()}>
						<Tabs.Trigger value="claims" class="gap-1.5">
							<LucideList />
							{m.claims()}
						</Tabs.Trigger>
						<Tabs.Trigger value="json" class="gap-1.5">
							<LucideBraces />
							JSON
						</Tabs.Trigger>
					</Tabs.List>
				</Tabs.Root>
				{#if previewData}
					<CopyButton labeled value={activeJson} label={m.copy_json()} />
				{/if}
			</div>

			<div
				role="region"
				aria-label={m.claims()}
				aria-busy={isLoading}
				class={cn(
					'rounded-md border transition-opacity duration-(--dur-base)',
					isLoading && previewData && 'opacity-50 delay-(--t-dim-delay)'
				)}
			>
				{#if errorMessage && !isLoading}
					<div class="flex flex-col items-center gap-1 px-6 py-10 text-center">
						<p class="text-heading">{m.couldnt_load_the_preview()}</p>
						<p class="text-control text-fg-muted max-w-sm">{errorMessage}</p>
					</div>
				{:else if !previewData}
					{#each Array.from({ length: 6 }, (_, i) => i) as i (i)}
						<div class="flex h-10 items-center gap-4 border-b px-4 last:border-b-0">
							<Skeleton class="h-3.5 w-20" />
							<Skeleton class="h-3.5 w-1/2" />
						</div>
					{/each}
				{:else if view === 'json'}
					<pre
						class="bg-surface-sunken text-mono-s rounded-[inherit] p-4 font-mono break-all whitespace-pre-wrap">{activeJson}</pre>
				{:else}
					<dl data-testid="preview-claims">
						{#each Object.entries(activeData) as [key, value] (key)}
							<div
								class="grid gap-1 border-b px-4 py-2.5 last:border-b-0 sm:grid-cols-[9rem_minmax(0,1fr)] sm:gap-4"
							>
								<dt class="text-mono-s text-fg-muted font-mono leading-6">{key}</dt>
								<dd class="text-control min-w-0 leading-6">{@render claimValue(key, value)}</dd>
							</div>
						{/each}
					</dl>
				{/if}
			</div>
		</div>
	</Sheet.Content>
</Sheet.Root>
