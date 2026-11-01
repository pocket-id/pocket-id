import { goto } from '$app/navigation';
import WebAuthnService from '#lib/services/webauthn-service.ts';
import unsavedChangesStore from '#lib/stores/unsaved-changes-store.svelte.ts';

const webauthnService = new WebAuthnService();

export async function signOut() {
	if (!unsavedChangesStore.hasChanges) {
		await webauthnService.logout();
	}

	// With unsaved changes the bar blocks this navigation and shakes instead
	goto('/login');
}
