<script lang="ts">
	import * as Field from '#lib/components/ui/field/index.ts';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { Input } from '#lib/components/ui/input/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { callbackUrlSchema } from '#lib/utils/zod-util.ts';
	import { LucideTrash2 } from '@lucide/svelte';
	import { tick, type Snippet } from 'svelte';
	import ChangedMarker from './changed-marker.svelte';

	let {
		label,
		description,
		urls = $bindable(),
		error = null,
		readonly = false,
		placeholder,
		emptyText,
		testIdPrefix,
		changed = false,
		onReset
	}: {
		label: string;
		description: string | Snippet;
		urls: string[];
		error?: string | null;
		readonly?: boolean;
		placeholder: string;
		emptyText: string;
		testIdPrefix: string;
		changed?: boolean;
		onReset?: () => void;
	} = $props();

	// Long lists fold behind "Show all" so the fields after the editor stay close
	const collapsedCount = 5;

	const id = $props.id();
	let draft = $state('');
	let draftError = $state<string | null>(null);
	let announcement = $state('');
	let showAll = $state(false);
	let input = $state<HTMLInputElement | null>(null);
	let list = $state<HTMLElement | null>(null);

	const visibleUrls = $derived(showAll ? urls : urls.slice(0, collapsedCount));

	function add() {
		const url = draft.trim();
		if (!url) return;
		if (urls.includes(url)) {
			draftError = m.this_url_is_already_in_the_list();
			return;
		}
		const result = callbackUrlSchema.safeParse(url);
		if (!result.success) {
			draftError = result.error.issues[0].message;
			return;
		}
		urls = [url, ...urls];
		announcement = m.added_url({ url });
		draft = '';
		input?.focus();
	}

	// Focus moves to the next row's remove button, or back to the input once the list is empty
	async function remove(index: number) {
		urls = urls.filter((_, i) => i !== index);
		await tick();
		const buttons = list?.querySelectorAll<HTMLButtonElement>('button[data-remove]');
		(buttons?.[Math.min(index, buttons.length - 1)] ?? input)?.focus();
	}
</script>

<Field.Field class="relative">
	<div>
		<div class="flex items-center gap-2">
			<Field.Label for={id}>{label}</Field.Label>
			{#if changed}
				<span class="bg-fg size-1.5 shrink-0 rounded-full"></span>
				<span class="sr-only">{m.changed_from_the_default()}</span>
			{/if}
		</div>
		<Field.Description>
			{#if typeof description === 'string'}{description}{:else}{@render description()}{/if}
		</Field.Description>
		{#if changed && onReset}
			<ChangedMarker {label} {onReset} />
		{/if}
	</div>
	{#if !readonly}
		<div class="flex gap-2 max-[400px]:flex-col">
			<Input
				{id}
				bind:ref={input}
				bind:value={draft}
				{placeholder}
				class="font-mono"
				inputmode="url"
				autocomplete="off"
				aria-invalid={!!draftError}
				data-testid="{testIdPrefix}-input"
				oninput={() => (draftError = null)}
				onkeydown={(e) => {
					if (e.key === 'Enter') {
						e.preventDefault();
						add();
					}
				}}
			/>
			<Button variant="secondary" disabled={!draft.trim()} onclick={add}>{m.add()}</Button>
		</div>
		{#if draftError}
			<Field.Error>{draftError}</Field.Error>
		{/if}
	{/if}
	<ul bind:this={list} aria-label={label} class="flex flex-col rounded-md border">
		{#each visibleUrls as url, i (url)}
			<li
				class="text-mono-s flex min-h-9 items-center gap-2 border-b py-1 ps-3 pe-1 font-mono last:border-b-0"
				data-testid="{testIdPrefix}-{i + 1}"
			>
				<span class="min-w-0 flex-1 break-all">{url}</span>
				{#if !readonly}
					<Button
						variant="ghost"
						size="icon-xs"
						data-remove
						aria-label={m.remove_url({ identifier: url })}
						onclick={() => remove(i)}
					>
						<LucideTrash2 />
					</Button>
				{/if}
			</li>
		{:else}
			<li class="text-control text-fg-muted flex min-h-9 items-center px-3">{emptyText}</li>
		{/each}
	</ul>
	{#if urls.length > collapsedCount}
		<Button variant="link" size="sm" class="w-fit!" onclick={() => (showAll = !showAll)}>
			{showAll ? m.show_less() : m.show_all_count({ count: urls.length })}
		</Button>
	{/if}
	<span class="sr-only" aria-live="polite">{announcement}</span>
	{#if error}
		<Field.Error>{error}</Field.Error>
	{/if}
</Field.Field>
