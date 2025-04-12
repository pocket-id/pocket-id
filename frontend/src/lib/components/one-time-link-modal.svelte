<script lang="ts">
	import { page } from '$app/state';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import Input from '$lib/components/ui/input/input.svelte';
	import Label from '$lib/components/ui/label/label.svelte';
	import * as Select from '$lib/components/ui/select/index.js';
	import { m } from '$lib/paraglide/messages';
	import UserService from '$lib/services/user-service';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import Qrcode from './qrcode/qrcode.svelte';

	let {
		userId = $bindable()
	}: {
		userId: string | null;
	} = $props();

	const userService = new UserService();

	let oneTimeLink: string | null = $state(null);
	let selectedExpiration: keyof typeof availableExpirations = $state(m.one_hour());

	let availableExpirations = {
		[m.one_hour()]: 60 * 60,
		[m.twelve_hours()]: 60 * 60 * 12,
		[m.one_day()]: 60 * 60 * 24,
		[m.one_week()]: 60 * 60 * 24 * 7,
		[m.one_month()]: 60 * 60 * 24 * 30
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
			<Dialog.Title>{m.login_code()}</Dialog.Title>
			<Dialog.Description
				>{m.create_a_login_code_to_sign_in_without_a_passkey_once()}</Dialog.Description
			>
		</Dialog.Header>
		{#if oneTimeLink === null}
			<div>
				<Label for="expiration">{m.expiration()}</Label>
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
				{m.generate_code()}
			</Button>
		{:else}
			<Label for="login-code" class="sr-only">{m.login_code()}</Label>
			<Input id="login-code" value={oneTimeLink} readonly />
			<Qrcode
				value={oneTimeLink}
				size={180}
				margin={0}
				color="#FFFFFF"
				backgroundColor="#00000000"
			/>
		{/if}
	</Dialog.Content>
</Dialog.Root>
