<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { m } from '$lib/paraglide/messages';
	import WebAuthnService from '$lib/services/webauthn-service';
	import type { Passkey } from '$lib/types/passkey.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { preventDefault } from '$lib/utils/event-util';
	import { toast } from 'svelte-sonner';

	let {
		passkey = $bindable(),
		callback
	}: {
		passkey: Passkey | null;
		callback?: () => void;
	} = $props();

	let name = $state('');

	$effect(() => {
		if (passkey) name = passkey.name;
	});

	const webauthnService = new WebAuthnService();

	function onOpenChange(open: boolean) {
		if (!open) {
			passkey = null;
		}
	}

	async function onSubmit() {
		await webauthnService
			.updateCredentialName(passkey!.id, name)
			.then(() => {
				passkey = null;
				toast.success(m.passkey_name_updated_successfully());
				callback?.();
			})
			.catch(axiosErrorToast);
	}
</script>

<Dialog.Root open={!!passkey} {onOpenChange}>
	<Dialog.Content class="max-w-md">
		<Dialog.Header>
			<Dialog.Title>{m.name_passkey()}</Dialog.Title>
			<Dialog.Description>{m.name_your_passkey_to_easily_identify_it_later()}</Dialog.Description>
		</Dialog.Header>
		<form onsubmit={preventDefault(onSubmit)}>
			<div class="grid items-center gap-4 sm:grid-cols-4">
				<Label for="name" class="sm:text-right">{m.name()}</Label>
				<Input id="name" bind:value={name} class="col-span-3" />
			</div>
			<Dialog.Footer class="mt-4">
				<Button type="submit">{m.save()}</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
