<script lang="ts">
	import { page } from '$app/state';
	import CopyToClipboard from '$lib/components/copy-to-clipboard.svelte';
	import Qrcode from '$lib/components/qrcode/qrcode.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import Label from '$lib/components/ui/label/label.svelte';
	import * as Select from '$lib/components/ui/select/index.js';
	import { Separator } from '$lib/components/ui/separator';
	import { m } from '$lib/paraglide/messages';
	import UserService from '$lib/services/user-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { mode } from 'mode-watcher';
	import { toast } from 'svelte-sonner';

	let {
		userId = $bindable()
	}: {
		userId: string | null;
	} = $props();

	const userService = new UserService();

	let oneTimeLink: string | null = $state(null);
	let code: string | null = $state(null);
	let selectedExpiration: keyof typeof availableExpirations = $state(m.one_hour());

	let availableExpirations = {
		[m.one_hour()]: 60 * 60,
		[m.twelve_hours()]: 60 * 60 * 12,
		[m.one_day()]: 60 * 60 * 24,
		[m.one_week()]: 60 * 60 * 24 * 7,
		[m.one_month()]: 60 * 60 * 24 * 30
	};

	async function createLoginCode() {
		try {
			const expiration = new Date(Date.now() + availableExpirations[selectedExpiration] * 1000);
			code = await userService.createOneTimeAccessToken(expiration, userId!);
			oneTimeLink = `${page.url.origin}/lc/${code}`;
		} catch (e) {
			axiosErrorToast(e);
		}
	}

	async function sendLoginCodeEmail() {
		try {
			const expiration = new Date(Date.now() + availableExpirations[selectedExpiration] * 1000);
			await userService.requestOneTimeAccessEmailAsAdmin(userId!, expiration);
			toast.success(m.login_code_email_success());
			onOpenChange(false);
		} catch (e) {
			axiosErrorToast(e);
		}
	}

	function onOpenChange(open: boolean) {
		if (!open) {
			oneTimeLink = null;
			code = null;
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
					type="single"
					value={Object.keys(availableExpirations)[0]}
					onValueChange={(v) => (selectedExpiration = v! as keyof typeof availableExpirations)}
				>
					<Select.Trigger id="expiration" class="h-9 w-full">
						{selectedExpiration}
					</Select.Trigger>
					<Select.Content>
						{#each Object.keys(availableExpirations) as key}
							<Select.Item value={key}>{key}</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			</div>
			<Dialog.Footer class="mt-2">
				{#if $appConfigStore.emailOneTimeAccessAsAdminEnabled}
					<Button
						onclick={() => sendLoginCodeEmail()}
						variant="secondary"
						disabled={!selectedExpiration}
					>
						{m.send_email()}
					</Button>
				{/if}
				<Button onclick={() => createLoginCode()} disabled={!selectedExpiration}
					>{m.show_code()}</Button
				>
			</Dialog.Footer>
		{:else}
			<div class="flex flex-col items-center gap-2">
				<CopyToClipboard value={code!}>
					<p class="text-3xl font-semibold">{code}</p>
				</CopyToClipboard>

				<div class="text-muted-foreground my-2 flex items-center justify-center gap-3">
					<Separator />
					<p class="text-xs text-nowrap">{m.or_visit()}</p>
					<Separator />
				</div>

				<Qrcode
					class="mb-2"
					value={oneTimeLink}
					size={180}
					color={mode.current === 'dark' ? '#FFFFFF' : '#000000'}
					backgroundColor={mode.current === 'dark' ? '#000000' : '#FFFFFF'}
				/>
				<CopyToClipboard value={oneTimeLink!}>
					<p data-testId="login-code-link">{oneTimeLink!}</p>
				</CopyToClipboard>
			</div>
		{/if}
	</Dialog.Content>
</Dialog.Root>
