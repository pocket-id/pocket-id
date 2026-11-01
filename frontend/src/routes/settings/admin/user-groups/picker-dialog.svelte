<script lang="ts" module>
	import { m } from '#lib/paraglide/messages.js';

	export const groupsConfirmLabel = (count: number) =>
		count === 0
			? m.add_to_groups()
			: count === 1
				? m.add_to_one_group()
				: m.add_to_count_groups({ count });
</script>

<script lang="ts" generics="T extends { id: string }">
	import { Button } from '#lib/components/ui/button/index.ts';
	import { Checkbox } from '#lib/components/ui/checkbox/index.ts';
	import * as Dialog from '#lib/components/ui/dialog/index.ts';
	import * as InputGroup from '#lib/components/ui/input-group/index.ts';
	import { Skeleton } from '#lib/components/ui/skeleton/index.ts';
	import { debounced } from '#lib/utils/debounce-util.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { LucideSearch } from '@lucide/svelte';
	import { untrack, type Snippet } from 'svelte';

	let {
		open = $bindable(),
		title,
		searchLabel,
		load,
		exclude = [],
		row,
		confirmLabel,
		onConfirm
	}: {
		open: boolean;
		title: string;
		searchLabel: string;
		load: (search: string) => Promise<T[]>;
		// Items that are already assigned; only what can still be added is listed
		exclude?: string[];
		row: Snippet<[T]>;
		confirmLabel: (count: number) => string;
		onConfirm: (ids: string[]) => Promise<void>;
	} = $props();

	let search = $state('');
	let searchInput = $state<HTMLInputElement | null>(null);
	let items = $state<T[]>();
	let selected = $state<string[]>([]);
	let busy = $state(false);
	// Taken when the dialog opens, so a parent that updates them after confirming does not change the dialog while it fades out
	let shownTitle = $state('');
	let excluded = $state<string[]>([]);

	const visible = $derived(items?.filter((item) => !excluded.includes(item.id)));

	async function refresh() {
		try {
			items = await load(search.trim());
		} catch (e) {
			axiosErrorToast(e);
		}
	}
	const onSearch = debounced(refresh, 300);

	// Every opening starts with an empty search and nothing checked
	$effect(() => {
		if (!open) return;
		untrack(() => {
			shownTitle = title;
			excluded = exclude;
			search = '';
			selected = [];
			items = undefined;
			refresh();
		});
	});

	function toggle(id: string, checked: boolean) {
		selected = checked ? [...selected, id] : selected.filter((s) => s !== id);
	}

	async function confirm() {
		busy = true;
		try {
			await onConfirm(selected);
			open = false;
		} catch (e) {
			axiosErrorToast(e);
		} finally {
			busy = false;
		}
	}
</script>

<Dialog.Root bind:open={() => open, (value) => !busy && (open = value)}>
	<Dialog.Content class="max-w-(--dialog-lg)">
		<Dialog.Header>
			<Dialog.Title>{shownTitle}</Dialog.Title>
		</Dialog.Header>
		<InputGroup.Root>
			<InputGroup.Input
				bind:ref={searchInput}
				bind:value={search}
				placeholder={searchLabel}
				aria-label={searchLabel}
				type="search"
				oninput={() => onSearch()}
			/>
			<InputGroup.Addon><LucideSearch /></InputGroup.Addon>
		</InputGroup.Root>
		<div
			role="group"
			aria-label={shownTitle}
			class="max-h-90 min-h-(--row-2line) overflow-y-auto rounded-md border"
		>
			{#if !visible}
				{#each [0, 1, 2] as i (i)}
					<div class="flex h-(--row-2line) items-center gap-3 border-b px-4 last:border-b-0">
						<Skeleton class="size-4.5 rounded-xs" />
						<Skeleton class="h-3.5 w-2/5" />
					</div>
				{/each}
			{:else if visible.length === 0}
				<div
					class="text-control text-fg-muted flex min-h-(--row-2line) flex-wrap items-center gap-x-2 px-4 wrap-anywhere"
				>
					{#if search.trim()}
						{m.no_matches_for_query({ query: search.trim() })}
						<Button
							variant="link"
							size="sm"
							class="px-0"
							onclick={() => {
								search = '';
								searchInput?.focus();
								refresh();
							}}>{m.clear_search()}</Button
						>
					{:else}
						{m.nothing_left_to_add()}
					{/if}
				</div>
			{:else}
				{#each visible as item (item.id)}
					<label
						class="hover:bg-fill-row-hover has-focus-visible:bg-fill-row-hover flex min-h-(--row-2line) cursor-pointer items-center gap-3 border-b px-4 py-2 transition-colors duration-(--dur-fast) last:border-b-0"
					>
						<Checkbox
							checked={selected.includes(item.id)}
							onCheckedChange={(checked) => toggle(item.id, checked)}
						/>
						{@render row(item)}
					</label>
				{/each}
			{/if}
		</div>
		<Dialog.Footer>
			<Button variant="ghost" disabled={busy} onclick={() => (open = false)}>{m.cancel()}</Button>
			<Button disabled={selected.length === 0} isLoading={busy} onclick={confirm}>
				{confirmLabel(selected.length)}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
