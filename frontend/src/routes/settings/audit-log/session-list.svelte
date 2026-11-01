<script lang="ts">
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
	import SessionRow from '#lib/components/session-row.svelte';
	import SessionSheet from '#lib/components/session-sheet.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import SessionService from '#lib/services/session-service.ts';
	import type { Session } from '#lib/types/session.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { sessionDeviceName } from '#lib/utils/session-util.ts';
	import { toast } from 'svelte-sonner';

	let { sessions = $bindable() }: { sessions: Session[] } = $props();

	const sessionService = new SessionService();

	let selected = $state<Session | null>(null);
	let sheetOpen = $state(false);

	const sorted = $derived([...sessions].sort((a, b) => Number(b.current) - Number(a.current)));
	const hasOtherSessions = $derived(sessions.some((session) => !session.current));

	function select(session: Session) {
		selected = session;
		sheetOpen = true;
	}

	function revoke(session: Session) {
		sheetOpen = false;
		openConfirmDialog({
			title: m.sign_out_device_question({ device: sessionDeviceName(session) }),
			message: m.sign_out_device_consequence(),
			confirm: {
				label: m.sign_out(),
				destructive: true,
				action: async () => {
					try {
						await sessionService.revoke(session.id);
						sessions = await sessionService.list();
						toast.success(m.device_signed_out());
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}

	function revokeOthers() {
		openConfirmDialog({
			title: m.sign_out_other_devices_question(),
			message: m.sign_out_other_devices_consequence(),
			confirm: {
				label: m.sign_out_other_devices(),
				destructive: true,
				action: async () => {
					try {
						await sessionService.revokeOthers();
						sessions = await sessionService.list();
						toast.success(m.signed_out_of_other_devices());
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}
</script>

<SettingsSection title={m.signed_in_devices()} description={m.signed_in_devices_description()}>
	{#snippet action()}
		{#if hasOtherSessions}
			<Button variant="secondary" size="sm" onclick={revokeOthers}>
				{m.sign_out_other_devices()}
			</Button>
		{/if}
	{/snippet}

	<!-- Collections inside a section are open rows with hairlines, so the shared row loses its own inset -->
	<div
		role="list"
		aria-label={m.signed_in_devices()}
		class="pt-2 *:*:min-h-[68px] *:*:px-0 *:shadow-[inset_0_-1px_0_var(--border)] *:last:shadow-none"
	>
		{#each sorted as session (session.id)}
			<div role="listitem">
				<SessionRow {session} onSelect={() => select(session)} onRevoke={() => revoke(session)} />
			</div>
		{/each}
	</div>
</SettingsSection>

<SessionSheet bind:open={sheetOpen} session={selected} onRevoke={revoke} />
