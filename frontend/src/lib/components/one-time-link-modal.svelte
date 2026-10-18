<script lang="ts">
	import { page } from '$app/state';
	import CopyField from '#lib/components/copy-field.svelte';
	import Qrcode from '#lib/components/qrcode/qrcode.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Dialog from '#lib/components/ui/dialog/index.ts';
	import * as Field from '#lib/components/ui/field/index.ts';
	import * as Select from '#lib/components/ui/select/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import UserService from '#lib/services/user-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
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
			code = await userService.createOneTimeAccessToken(
				userId!,
				availableExpirations[selectedExpiration]
			);
			oneTimeLink = `${page.url.origin}/lc/${code}`;
		} catch (e) {
			axiosErrorToast(e);
		}
	}

	async function sendLoginCodeEmail() {
		try {
			await userService.requestOneTimeAccessEmailAsAdmin(
				userId!,
				availableExpirations[selectedExpiration]
			);
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
			<Field.Field>
				<Field.Label for="expiration">{m.expiration()}</Field.Label>
				<Select.Root
					type="single"
					value={Object.keys(availableExpirations)[0]}
					onValueChange={(v) => (selectedExpiration = v! as keyof typeof availableExpirations)}
				>
					<Select.Trigger id="expiration" class="w-full">
						{selectedExpiration}
					</Select.Trigger>
					<Select.Content>
						{#each Object.keys(availableExpirations) as key (key)}
							<Select.Item value={key}>{key}</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			</Field.Field>
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
			<div class="flex flex-col gap-4">
				<CopyField label={m.code()} value={code!} copyLabel={m.copy_code()} />
				<CopyField
					label={m.link()}
					value={oneTimeLink}
					copyLabel={m.copy_link()}
					testId="login-code-link"
				/>
				<Qrcode class="self-center" value={oneTimeLink} size={160} />
			</div>
		{/if}
	</Dialog.Content>
</Dialog.Root>
