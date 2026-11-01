<script lang="ts">
	import { goto } from '$app/navigation';
	import * as Avatar from '#lib/components/ui/avatar/index.ts';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.ts';
	import * as Tooltip from '#lib/components/ui/tooltip/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import userStore from '#lib/stores/user-store.ts';
	import { cachedProfilePicture } from '#lib/utils/cached-image-util.ts';
	import { setModeInstantly } from '#lib/utils/color-theme-util.ts';
	import { signOut } from '#lib/utils/sign-out-util.ts';
	import { cn } from '#lib/utils/style.ts';
	import { IdCard, LogOut, SunMoon } from '@lucide/svelte';
	import { userPrefersMode } from 'mode-watcher';

	let {
		variant,
		version
	}: {
		// row: the sidebar's account row; rail: the avatar at the foot of the rail; avatar: the phone top bar
		variant: 'row' | 'rail' | 'avatar';
		version?: string;
	} = $props();

	let open = $state(false);
	let jumping = $state(false);

	const appearanceLabels = { light: m.light(), dark: m.dark(), system: m.system() };
</script>

{#snippet appearanceOptions()}
	<DropdownMenu.RadioGroup
		value={userPrefersMode.current}
		onValueChange={(v) => setModeInstantly(v as 'light' | 'dark' | 'system')}
	>
		<DropdownMenu.RadioItem value="light">{m.light()}</DropdownMenu.RadioItem>
		<DropdownMenu.RadioItem value="dark">{m.dark()}</DropdownMenu.RadioItem>
		<DropdownMenu.RadioItem value="system">{m.system()}</DropdownMenu.RadioItem>
	</DropdownMenu.RadioGroup>
{/snippet}

{#if $userStore}
	<DropdownMenu.Root
		bind:open
		onOpenChange={(o) => {
			if (o) jumping = false;
		}}
	>
		<Tooltip.Root disabled={open || variant === 'avatar'}>
			<Tooltip.Trigger>
				{#snippet child({ props: tooltipProps })}
					<DropdownMenu.Trigger {...tooltipProps}>
						{#snippet child({ props })}
							<button
								{...props}
								type="button"
								aria-label={variant === 'row' ? undefined : m.account_menu()}
								class={cn(
									'focus-visible:outline-ring flex items-center rounded-sm outline-offset-[-2px] focus-visible:outline-2',
									variant === 'row' &&
										'text-fg hover:bg-fill-hover data-[state=open]:bg-fill-selected h-9 w-full gap-2.5 px-2.5 text-left text-sm transition-colors duration-(--dur-fast) active:scale-(--press-scale)',
									variant === 'rail' &&
										'hover:bg-fill-hover data-[state=open]:bg-fill-selected size-9 justify-center',
									variant === 'avatar' && 'rounded-full'
								)}
							>
								<Avatar.Root class={variant === 'avatar' ? 'size-8' : 'size-5'}>
									<Avatar.Image src={cachedProfilePicture.getUrl($userStore.id)} alt="" />
									<Avatar.Fallback>{$userStore.firstName?.[0]}</Avatar.Fallback>
								</Avatar.Root>
								{#if variant === 'row'}
									<span class="min-w-0 truncate">{$userStore.displayName}</span>
								{/if}
							</button>
						{/snippet}
					</DropdownMenu.Trigger>
				{/snippet}
			</Tooltip.Trigger>
			<Tooltip.Content side="right">
				{$userStore.displayName}{#if version}<span class="text-inverse-muted">
						· {version}</span
					>{/if}
			</Tooltip.Content>
		</Tooltip.Root>

		<DropdownMenu.Content
			class="bg-popover w-60"
			side={variant === 'avatar' ? 'bottom' : 'top'}
			align={variant === 'avatar' ? 'end' : 'start'}
			data-jumping={jumping || undefined}
		>
			<div class="flex items-center gap-3 px-2 py-2">
				<Avatar.Root class="size-10">
					<Avatar.Image src={cachedProfilePicture.getUrl($userStore.id)} alt="" />
					<Avatar.Fallback>{$userStore.firstName?.[0]}</Avatar.Fallback>
				</Avatar.Root>
				<div class="flex min-w-0 flex-col">
					<span class="truncate text-[13px] font-medium">{$userStore.displayName}</span>
					<span class="text-fg-muted truncate text-xs">{$userStore.email}</span>
				</div>
			</div>
			<DropdownMenu.Separator />
			<DropdownMenu.Group>
				<DropdownMenu.Item
					onclick={() => {
						jumping = true;
						goto('/settings/account');
					}}
				>
					<IdCard />
					{m.my_account()}
				</DropdownMenu.Item>
				{#if variant !== 'avatar'}
					<DropdownMenu.Sub>
						<DropdownMenu.SubTrigger>
							<SunMoon />
							<span class="flex flex-col">
								{m.appearance()}
								<span class="text-fg-muted text-xs"
									>{appearanceLabels[userPrefersMode.current]}</span
								>
							</span>
						</DropdownMenu.SubTrigger>
						<DropdownMenu.SubContent class="bg-popover">
							{@render appearanceOptions()}
						</DropdownMenu.SubContent>
					</DropdownMenu.Sub>
				{/if}
			</DropdownMenu.Group>
			<!-- A phone has no room beside the menu for a submenu, so the options sit inline -->
			{#if variant === 'avatar'}
				<DropdownMenu.Separator />
				<DropdownMenu.Group>
					<DropdownMenu.GroupHeading>{m.appearance()}</DropdownMenu.GroupHeading>
					{@render appearanceOptions()}
				</DropdownMenu.Group>
			{/if}
			<DropdownMenu.Separator />
			<DropdownMenu.Item onclick={signOut}>
				<LogOut />
				{m.sign_out()}
			</DropdownMenu.Item>
		</DropdownMenu.Content>
	</DropdownMenu.Root>
{/if}
