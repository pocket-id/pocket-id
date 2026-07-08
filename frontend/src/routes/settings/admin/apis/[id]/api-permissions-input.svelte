<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { m } from '$lib/paraglide/messages';
	import type { ApiPermissionInput } from '$lib/types/api.type';
	import { LucideMinus, LucidePlus } from '@lucide/svelte';

	let { permissions = $bindable() }: { permissions: ApiPermissionInput[] } = $props();

	const limit = 100;
</script>

<div class="flex flex-col gap-y-3">
	{#each permissions as _, i}
		<div class="flex flex-col gap-2 sm:flex-row sm:items-center">
			<Input
				class="font-mono sm:w-1/3"
				placeholder={m.api_permission_key()}
				bind:value={permissions[i].key}
			/>
			<Input class="sm:w-1/4" placeholder={m.name()} bind:value={permissions[i].name} />
			<Input placeholder={m.description()} bind:value={permissions[i].description} />
			<Button
				variant="outline"
				size="sm"
				aria-label={m.delete()}
				onclick={() => (permissions = permissions.filter((_, index) => index !== i))}
			>
				<LucideMinus class="size-4" />
			</Button>
		</div>
	{/each}
</div>
{#if permissions.length < limit}
	<Button
		class="mt-3"
		variant="secondary"
		size="sm"
		onclick={() => (permissions = [...permissions, { key: '', name: '', description: '' }])}
	>
		<LucidePlus class="mr-1 size-4" />
		{permissions.length === 0 ? m.add_permission() : m.add_another()}
	</Button>
{/if}
