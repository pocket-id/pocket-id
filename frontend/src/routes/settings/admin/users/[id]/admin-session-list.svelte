<script lang="ts">
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
	import SessionRow from '#lib/components/session-row.svelte';
	import SessionSheet from '#lib/components/session-sheet.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import { Button } from '@pocket-id/ui/button';
	import { m } from '#lib/paraglide/messages.js';
	import SessionService from '#lib/services/session-service.ts';
	import type { Session } from '#lib/types/session.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { sessionDeviceName } from '#lib/utils/session-util.ts';
	import { toast } from 'svelte-sonner';

	let {
		userId,
		firstName,
		sessions = $bindable()
	}: {
		userId: string;
		firstName: string;
		sessions: Session[];
	} = $props();

	const sessionService = new SessionService();

	let selected = $state<Session | null>(null);
	let sheetOpen = $state(false);

	function select(session: Session) {
		selected = session;
		sheetOpen = true;
	}

	function revoke(session: Session) {
		sheetOpen = false;
		openConfirmDialog({
			title: m.sign_out_device_question({ device: sessionDeviceName(session) }),
			message: m.sign_out_user_device_consequence({ name: firstName }),
			confirm: {
				label: m.sign_out(),
				destructive: true,
				action: async () => {
					try {
						await sessionService.revokeForUser(userId, session.id);
						sessions = await sessionService.listForUser(userId);
						toast.success(m.device_signed_out());
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}

	function revokeAll() {
		openConfirmDialog({
			title: m.sign_out_user_everywhere_question({ name: firstName }),
			message: m.sign_out_user_everywhere_consequence({ name: firstName }),
			confirm: {
				label: m.sign_out_everywhere(),
				destructive: true,
				action: async () => {
					try {
						await sessionService.revokeAllForUser(userId);
						sessions = await sessionService.listForUser(userId);
						toast.success(m.user_signed_out_everywhere({ name: firstName }));
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}
</script>

<SettingsSection title={m.signed_in_devices()}>
	{#snippet action()}
		{#if sessions.length > 0}
			<Button variant="secondary" size="sm" onclick={revokeAll}>{m.sign_out_everywhere()}</Button>
		{/if}
	{/snippet}
	<div class="border-t">
		{#each sessions as session (session.id)}
			<div class="border-b">
				<SessionRow {session} onSelect={() => select(session)} onRevoke={() => revoke(session)} />
			</div>
		{:else}
			<p class="text-control text-fg-muted flex min-h-(--row-2line) items-center border-b px-4">
				{m.user_has_no_sessions({ name: firstName })}
			</p>
		{/each}
	</div>
</SettingsSection>

<SessionSheet bind:open={sheetOpen} session={selected} onRevoke={revoke} />
