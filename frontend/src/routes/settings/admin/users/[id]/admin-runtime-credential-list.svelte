<script lang="ts">
	import { openConfirmDialog } from '$lib/components/confirm-dialog';
	import RuntimeCredentialList from '$lib/components/runtime-credential-list.svelte';
	import { m } from '$lib/paraglide/messages';
	import RuntimeCredentialService from '$lib/services/runtime-credential-service';
	import type { RuntimeCredential } from '$lib/types/runtime-credential.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { toast } from 'svelte-sonner';

	let {
		userId,
		credentials = $bindable()
	}: {
		userId: string;
		credentials: RuntimeCredential[];
	} = $props();

	const runtimeCredentialService = new RuntimeCredentialService();

	function revokeCredential(credential: RuntimeCredential) {
		openConfirmDialog({
			title: m.revoke_runtime_credential_name({ credentialName: credential.name }),
			message: m.revoke_runtime_credential_description(),
			confirm: {
				label: m.revoke(),
				destructive: true,
				action: async () => {
					try {
						await runtimeCredentialService.revokeForUser(userId, credential.id);
						credentials = await runtimeCredentialService.listForUser(userId);
						toast.success(m.runtime_credential_revoked_successfully());
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}
</script>

<RuntimeCredentialList {credentials} onRevoke={revokeCredential} />
