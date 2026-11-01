<script lang="ts">
	import * as Avatar from '#lib/components/ui/avatar/index.ts';
	import type { User } from '#lib/types/user.type.ts';
	import { cachedProfilePicture } from '#lib/utils/cached-image-util.ts';
	import { cn } from '#lib/utils/style.ts';

	let {
		user,
		class: className
	}: { user: Pick<User, 'id' | 'firstName' | 'lastName'>; class?: string } = $props();
</script>

<Avatar.Root class={cn('size-10', className)}>
	<Avatar.Image src={cachedProfilePicture.getUrl(user.id)} alt="" />
	<Avatar.Fallback class="uppercase">{user.firstName?.[0]}{user.lastName?.[0]}</Avatar.Fallback>
</Avatar.Root>
