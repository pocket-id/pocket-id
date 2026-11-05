<script lang="ts">
	import * as Avatar from '@pocket-id/ui/avatar';
	import type { User } from '#lib/types/user.type.ts';
	import { cachedProfilePicture } from '#lib/utils/cached-asset-util.ts';
	import { cn } from '@pocket-id/ui/utils';

	let {
		user,
		class: className
	}: { user: Pick<User, 'id' | 'firstName' | 'lastName'>; class?: string } = $props();
</script>

<Avatar.Root class={cn('size-10', className)}>
	<Avatar.Image src={cachedProfilePicture.getUrl(user.id)} alt="" />
	<Avatar.Fallback class="uppercase">{user.firstName?.[0]}{user.lastName?.[0]}</Avatar.Fallback>
</Avatar.Root>
