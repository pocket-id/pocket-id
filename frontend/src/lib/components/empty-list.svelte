<script lang="ts">
	import type { Snippet } from 'svelte';
	import * as Table from './ui/table/index.ts';

	let {
		columns,
		title,
		description,
		action,
		avatar = true
	}: {
		// The list's own column headers, so the empty page already has the shape the list will have
		columns: string[];
		title: string;
		description: Snippet;
		action: Snippet;
		avatar?: boolean;
	} = $props();

	const rows = [
		{ name: 'w-30', detail: 'w-50', value: 'w-6' },
		{ name: 'w-24', detail: 'w-42', value: 'w-4' },
		{ name: 'w-34', detail: 'w-48', value: 'w-5' }
	];
</script>

<div class="mt-10">
	<div aria-hidden="true" class="pointer-events-none mask-b-from-30%">
		<Table.Root>
			<Table.Header>
				<Table.Row>
					{#each columns as column, i (column)}
						<Table.Head class={i > 0 ? 'w-[30%] max-sm:hidden' : 'max-sm:rounded-r-sm'}
							>{column}</Table.Head
						>
					{/each}
				</Table.Row>
			</Table.Header>
			<Table.Body>
				{#each rows as row, r (r)}
					<Table.Row>
						<Table.Cell>
							<div class="flex items-center gap-3">
								{#if avatar}
									<div class="bg-fill-selected size-7 rounded-sm"></div>
								{/if}
								<div class="flex flex-col gap-1.5">
									<div class="bg-fill-selected h-2 rounded-xs {row.name}"></div>
									<div class="bg-fill-selected h-1.5 rounded-xs {row.detail}"></div>
								</div>
							</div>
						</Table.Cell>
						{#each columns.slice(1) as column (column)}
							<Table.Cell class="max-sm:hidden">
								<div class="bg-fill-selected h-2 rounded-xs {row.value}"></div>
							</Table.Cell>
						{/each}
					</Table.Row>
				{/each}
			</Table.Body>
		</Table.Root>
	</div>
	<div class="-mt-6 flex flex-col items-center gap-2 pb-12 text-center">
		<h2 class="text-title">{title}</h2>
		<p class="text-body text-fg-muted max-w-[380px]">{@render description()}</p>
		<div class="mt-4 flex flex-wrap items-center justify-center gap-4">{@render action()}</div>
	</div>
</div>
