<script lang="ts">
	import * as Avatar from '#lib/components/ui/avatar/index.ts';
	import type { User } from '#lib/types/user.type.ts';
	import { cachedProfilePicture } from '#lib/utils/cached-image-util.ts';
	import type { Snippet } from 'svelte';

	let { user, children }: { user: User; children?: Snippet } = $props();

	const fullName = $derived(
		user.displayName || [user.firstName, user.lastName].filter(Boolean).join(' ').trim()
	);
</script>

<div
	class="bg-surface flex min-h-15 w-full items-center gap-3 rounded-md border px-4 py-3 text-start"
>
	<Avatar.Root class="size-8">
		<Avatar.Image src={cachedProfilePicture.getUrl(user.id)} alt="" />
		<Avatar.Fallback>{user.firstName?.[0]}</Avatar.Fallback>
	</Avatar.Root>
	<div class="flex min-w-0 flex-1 flex-col gap-0.5">
		<p class="text-heading truncate font-medium">{fullName || user.email}</p>
		{#if fullName && user.email}
			<p class="text-control text-fg-muted truncate">{user.email}</p>
		{/if}
	</div>
	{@render children?.()}
</div>
