<script lang="ts">
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Dialog from '#lib/components/ui/dialog/index.ts';
	import * as Field from '#lib/components/ui/field/index.ts';
	import { Input } from '#lib/components/ui/input/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import WebAuthnService from '#lib/services/webauthn-service.ts';
	import type { Passkey } from '#lib/types/passkey.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { preventDefault } from '#lib/utils/event-util.ts';
	import { toast } from 'svelte-sonner';

	let {
		passkey = $bindable(),
		callback
	}: {
		passkey: Passkey | null;
		callback?: () => void;
	} = $props();

	let name = $state('');
	let input = $state<HTMLInputElement | null>(null);

	$effect.pre(() => {
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
			.updateCredentialName(passkey!.id, name.trim())
			.then(() => {
				passkey = null;
				toast.success(m.passkey_name_updated_successfully());
				callback?.();
			})
			.catch(axiosErrorToast);
	}
</script>

<Dialog.Root open={!!passkey} {onOpenChange}>
	<Dialog.Content
		class="max-w-(--dialog-sm)"
		onOpenAutoFocus={(e) => {
			e.preventDefault();
			input?.focus();
			input?.select();
		}}
	>
		<Dialog.Header>
			<Dialog.Title>{m.name_passkey()}</Dialog.Title>
			<Dialog.Description>{m.name_your_passkey_to_easily_identify_it_later()}</Dialog.Description>
		</Dialog.Header>
		<form onsubmit={preventDefault(onSubmit)}>
			<Field.Field>
				<Field.Label for="name">{m.name()}</Field.Label>
				<Input id="name" bind:ref={input} bind:value={name} maxlength={50} />
			</Field.Field>
			<Dialog.Footer class="mt-4">
				<Button type="submit" disabled={!name.trim()}>{m.save()}</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
