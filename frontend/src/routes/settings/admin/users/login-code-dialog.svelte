<script lang="ts" module>
	import { m } from '#lib/paraglide/messages.js';

	const HOUR = 60 * 60;

	export const expirations = () => [
		{ label: m.one_hour(), seconds: HOUR },
		{ label: m.twelve_hours(), seconds: HOUR * 12 },
		{ label: m.one_day(), seconds: HOUR * 24 },
		{ label: m.one_week(), seconds: HOUR * 24 * 7 },
		{ label: m.one_month(), seconds: HOUR * 24 * 30 }
	];
</script>

<script lang="ts">
	import { page } from '$app/state';
	import CopyField from '#lib/components/copy-field.svelte';
	import Qrcode from '#lib/components/qrcode/qrcode.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Dialog from '#lib/components/ui/dialog/index.ts';
	import * as Field from '#lib/components/ui/field/index.ts';
	import * as Select from '#lib/components/ui/select/index.ts';
	import { getLocale } from '#lib/paraglide/runtime.js';
	import UserService from '#lib/services/user-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { User } from '#lib/types/user.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { LucideInfo } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';

	let { user = $bindable() }: { user: User | null } = $props();

	const userService = new UserService();

	let seconds = $state(HOUR);
	let code = $state<string | null>(null);
	let expiresAt = $state<Date>();
	let busy = $state<'code' | 'email' | null>(null);
	let open = $derived(!!user);

	const name = $derived(user?.firstName || user?.displayName || '');
	const link = $derived(code && `${page.url.origin}/lc/${code}`);
	const expiration = $derived(expirations().find((e) => e.seconds === seconds)!);

	async function createCode() {
		busy = 'code';
		try {
			code = await userService.createOneTimeAccessToken(user!.id, seconds);
			expiresAt = new Date(Date.now() + seconds * 1000);
		} catch (e) {
			axiosErrorToast(e);
		} finally {
			busy = null;
		}
	}

	async function sendEmail() {
		busy = 'email';
		try {
			await userService.requestOneTimeAccessEmailAsAdmin(user!.id, seconds);
			toast.success(m.login_code_sent_to_email({ email: user!.email ?? '' }));
			close();
		} catch (e) {
			axiosErrorToast(e);
		} finally {
			busy = null;
		}
	}

	function close() {
		open = false;
	}

	// The content stays until the exit animation ends, so the dialog does not fall back to the first step while it fades out
	function reset() {
		user = null;
		code = null;
		seconds = HOUR;
	}
</script>

<Dialog.Root bind:open onOpenChangeComplete={(o) => !o && reset()}>
	<Dialog.Content class={code ? 'max-w-(--dialog-lg)' : undefined}>
		<Dialog.Header>
			<Dialog.Title>{m.login_code_for_name({ name: user?.displayName ?? '' })}</Dialog.Title>
			<Dialog.Description>{m.login_code_description({ name })}</Dialog.Description>
		</Dialog.Header>

		{#if !code}
			<Field.Field>
				<Field.Label for="login-code-expiration">{m.expires_after()}</Field.Label>
				<Select.Root
					type="single"
					value={String(seconds)}
					onValueChange={(v) => v && (seconds = Number(v))}
				>
					<Select.Trigger id="login-code-expiration" class="w-full">
						{expiration.label}
					</Select.Trigger>
					<Select.Content>
						{#each expirations() as option (option.seconds)}
							<Select.Item value={String(option.seconds)}>{option.label}</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			</Field.Field>
			<Dialog.Footer>
				<Button variant="ghost" onclick={close}>{m.cancel()}</Button>
				{#if $appConfigStore.emailOneTimeAccessAsAdminEnabled && user?.email}
					<Button
						variant="secondary"
						isLoading={busy === 'email'}
						disabled={!!busy}
						onclick={sendEmail}
					>
						{m.send_by_email()}
					</Button>
				{/if}
				<Button isLoading={busy === 'code'} disabled={!!busy} onclick={createCode}>
					{m.create_code()}
				</Button>
			</Dialog.Footer>
		{:else}
			<CopyField label={m.code()} value={code} copyLabel={m.copy_code()} />
			<CopyField
				label={m.link()}
				value={link!}
				copyLabel={m.copy_link()}
				testId="login-code-link"
			/>
			<div class="flex items-center gap-6">
				<Qrcode value={link} size={140} class="shrink-0 rounded-sm bg-white p-2" />
				<div class="text-caption flex flex-col gap-1">
					<p class="text-fg tabular-nums">
						{m.expires_in_duration_at_time({
							duration: expiration.label,
							time: expiresAt!.toLocaleTimeString(getLocale(), {
								hour: '2-digit',
								minute: '2-digit'
							})
						})}
					</p>
				</div>
			</div>
			<p class="text-caption text-fg-muted flex items-start gap-1.5">
				<LucideInfo class="mt-[3px] size-3 shrink-0" />
				{m.login_code_share_note({ name })}
			</p>
			<Dialog.Footer>
				<Button onclick={close}>{m.done()}</Button>
			</Dialog.Footer>
		{/if}
	</Dialog.Content>
</Dialog.Root>
