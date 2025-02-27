<script lang="ts">
	import { page } from '$app/stores';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import Input from '$lib/components/ui/input/input.svelte';
	import Label from '$lib/components/ui/label/label.svelte';
	import * as Select from '$lib/components/ui/select/index.js';
	import UserService from '$lib/services/user-service';
	import { axiosErrorToast } from '$lib/utils/error-util';

	let {
		userId = $bindable(),
		show = $bindable()
	}: {
		userId: string | null;
		show: boolean | null;
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
			const token = await userService.createOneTimeAccessToken(expiration, userId);
			oneTimeLink = `${$page.url.origin}/login/${token}`;
		} catch (e) {
			axiosErrorToast(e);
		}
	}

	function onOpenChange(open: boolean) {
		if (!open) {
			oneTimeLink = null;
			userId = null;
			show = null;
		}
	}
</script>

<Dialog.Root open={!!userId || show} {onOpenChange}>
	<Dialog.Content class="max-w-md">
		<Dialog.Header>
			<Dialog.Title>One Time Link</Dialog.Title>
			<Dialog.Description
				>Use this link to sign in once and add a passkey from a new device.</Dialog.Description
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
				Generate Link
			</Button>
		{:else}
			<Label for="one-time-link" class="sr-only">One Time Link</Label>
			<Input id="one-time-link" value={oneTimeLink} readonly />
		{/if}
	</Dialog.Content>
</Dialog.Root>
