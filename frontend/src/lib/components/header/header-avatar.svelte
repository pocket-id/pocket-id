<script lang="ts">
	import * as Avatar from '$lib/components/ui/avatar';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import { m } from '$lib/paraglide/messages';
	import { setLocale } from '$lib/paraglide/runtime';
	import WebAuthnService from '$lib/services/webauthn-service';
	import userStore from '$lib/stores/user-store';
	import { LucideLogOut, LucideUser } from 'lucide-svelte';

	const webauthnService = new WebAuthnService();

	async function logout() {
		await webauthnService.logout();
		window.location.reload();
	}
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger
		><Avatar.Root class="h-9 w-9">
			<Avatar.Image src="/api/users/me/profile-picture.png" />
		</Avatar.Root></DropdownMenu.Trigger
	>
	<DropdownMenu.Content class="min-w-40" align="start">
		<DropdownMenu.Label class="font-normal">
			<div class="flex flex-col space-y-1">
				<p class="text-sm font-medium leading-none">
					{$userStore?.firstName}
					{$userStore?.lastName}
				</p>
				<p class="text-muted-foreground text-xs leading-none">{$userStore?.email}</p>
			</div>
		</DropdownMenu.Label>
		<DropdownMenu.Separator />
		<DropdownMenu.Group>
			<DropdownMenu.Item href="/settings/account"
				><LucideUser class="mr-2 h-4 w-4" /> {m.my_account()}</DropdownMenu.Item
			>
			<DropdownMenu.Item on:click={logout}
				><LucideLogOut class="mr-2 h-4 w-4" /> {m.logout()}</DropdownMenu.Item
			>
		</DropdownMenu.Group>
		<DropdownMenu.Separator />
		<DropdownMenu.Label>{m.language()}</DropdownMenu.Label>
		<DropdownMenu.Item on:click={() => setLocale('en')}>{m.english()}</DropdownMenu.Item>
		<DropdownMenu.Item on:click={() => setLocale('nl')}>{m.dutch()}</DropdownMenu.Item>
	</DropdownMenu.Content>
</DropdownMenu.Root>
