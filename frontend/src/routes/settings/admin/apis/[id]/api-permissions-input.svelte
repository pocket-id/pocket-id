<script lang="ts">
	import { Button } from '#lib/components/ui/button/index.ts';
	import { Input } from '#lib/components/ui/input/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import type { ApiPermissionInput } from '#lib/types/api.type.ts';
	import { LucidePlus, LucideX } from '@lucide/svelte';
	import { tick } from 'svelte';

	let { permissions = $bindable() }: { permissions: ApiPermissionInput[] } = $props();

	const limit = 100;

	let editor = $state<HTMLElement>();

	// The empty state's button disappears once a row exists, so focus moves to the new row's key instead of falling back to the page
	async function add() {
		permissions = [...permissions, { key: '', name: '', description: '' }];
		await tick();
		const keyInputs = editor?.querySelectorAll<HTMLInputElement>('[data-permission-key]');
		keyInputs?.item(keyInputs.length - 1).focus();
	}
</script>

<div class="flex flex-col gap-2">
	<div bind:this={editor} class="rounded-md border">
		{#if permissions.length === 0}
			<div class="text-control text-fg-muted flex min-h-(--row-2line) items-center gap-3 px-4">
				{m.no_permissions()}
				<Button variant="ghost" size="sm" onclick={add}>
					<LucidePlus />
					{m.add_permission()}
				</Button>
			</div>
		{:else}
			<div
				aria-hidden="true"
				class="text-caption text-fg-muted grid grid-cols-[1fr_1fr_1.4fr_var(--h-xs)] gap-2 px-3 pt-2.5 font-medium max-md:hidden"
			>
				<span>{m.key()}</span>
				<span>{m.name()}</span>
				<span>{m.description()}</span>
			</div>
			{#each permissions as permission, i (permission)}
				<div
					class="grid grid-cols-[1fr_var(--h-xs)] items-center gap-2 border-b px-3 py-2 md:grid-cols-[1fr_1fr_1.4fr_var(--h-xs)]"
				>
					<Input
						class="h-(--h-sm) font-mono"
						placeholder={m.key()}
						aria-label={m.key()}
						data-permission-key
						bind:value={permission.key}
					/>
					<Input
						class="h-(--h-sm) max-md:col-start-1"
						placeholder={m.name()}
						aria-label={m.name()}
						bind:value={permission.name}
					/>
					<Input
						class="h-(--h-sm) max-md:col-start-1"
						placeholder={m.description()}
						aria-label={m.description()}
						bind:value={permission.description}
					/>
					<Button
						variant="ghost"
						size="icon-xs"
						class="max-md:col-start-2 max-md:row-start-1"
						aria-label={m.remove_permission_name({ name: permission.key || String(i + 1) })}
						onclick={() => (permissions = permissions.filter((_, index) => index !== i))}
					>
						<LucideX />
					</Button>
				</div>
			{/each}
			<div class="px-2 py-1.5">
				<Button variant="ghost" size="sm" disabled={permissions.length >= limit} onclick={add}>
					<LucidePlus />
					{m.add_another()}
				</Button>
			</div>
		{/if}
	</div>
	<p class="text-caption text-fg-muted">{m.api_permissions_helper()}</p>
</div>
