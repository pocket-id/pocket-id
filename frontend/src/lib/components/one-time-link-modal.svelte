<script lang="ts">
	import { page } from '$app/state';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import Input from '$lib/components/ui/input/input.svelte';
	import Label from '$lib/components/ui/label/label.svelte';
	import * as Select from '$lib/components/ui/select/index.js';
	import UserService from '$lib/services/user-service';
	import { axiosErrorToast } from '$lib/utils/error-util';

	let {
		userId = $bindable()
	}: {
		userId: string | null;
	} = $props();

	const userService = new UserService();

	let oneTimeLink: string | null = $state(null);
	let selectedExpiration: keyof typeof availableExpirations = $state('1 hour');

	let availableExpirations = {
		'1 hour': 60 * 60,
		'12 hours': 60 * 60 * 12,
		'1 day': 60 * 60 * 24,
		'1 week': 60 * 60 * 24 * 7,
		'1 month': 60 * 60 * 24 * 30
	};

	async function createOneTimeAccessToken() {
		try {
			const expiration = new Date(Date.now() + availableExpirations[selectedExpiration] * 1000);
			const token = await userService.createOneTimeAccessToken(expiration, userId!);
			oneTimeLink = `${page.url.origin}/lc/${token}`;
		} catch (e) {
			axiosErrorToast(e);
		}
	}

	function onOpenChange(open: boolean) {
		if (!open) {
			oneTimeLink = null;
			userId = null;
		}
	}
</script>

<Dialog.Root open={!!userId} {onOpenChange}>
	<Dialog.Content class="max-w-md">
		<Dialog.Header>
			<Dialog.Title>Login Code</Dialog.Title>
			<Dialog.Description
				>Create a login code that the user can use to sign in without a passkey once.</Dialog.Description
			>
		</Dialog.Header>
		{#if oneTimeLink === null}
			<div>
				<Label for="expiration">Expiration</Label>
				<Select.Root
					selected={{
						label: Object.keys(availableExpirations)[0],
						value: Object.keys(availableExpirations)[0]
					}}
					onSelectedChange={(v) =>
						(selectedExpiration = v!.value as keyof typeof availableExpirations)}
				>
					<Select.Trigger class="h-9 ">
						<Select.Value>{selectedExpiration}</Select.Value>
					</Select.Trigger>
					<Select.Content>
						{#each Object.keys(availableExpirations) as key}
							<Select.Item value={key}>{key}</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			</div>
			<Button onclick={() => createOneTimeAccessToken()} disabled={!selectedExpiration}>
				Generate Code
			</Button>
		{:else}
			<Label for="login-code" class="sr-only">Login Code</Label>
			<Input id="login-code" value={oneTimeLink} readonly />
		{/if}
	</Dialog.Content>
</Dialog.Root>
