<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import * as Field from '$lib/components/ui/field';
	import { Input } from '$lib/components/ui/input';
	import { m } from '$lib/paraglide/messages';
	import RuntimeCredentialService from '$lib/services/runtime-credential-service';
	import type { RuntimeCredential } from '$lib/types/runtime-credential.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { preventDefault } from '$lib/utils/event-util';
	import { toast } from 'svelte-sonner';

	let {
		credential = $bindable(),
		callback
	}: {
		credential: RuntimeCredential | null;
		callback?: () => void;
	} = $props();

	let name = $state('');
	const runtimeCredentialService = new RuntimeCredentialService();

	$effect(() => {
		if (credential) name = credential.name;
	});

	function onOpenChange(open: boolean) {
		if (!open) credential = null;
	}

	async function onSubmit() {
		if (!credential || name.trim().length === 0 || name.trim().length > 50) return;
		await runtimeCredentialService
			.updateName(credential.id, name.trim())
			.then(() => {
				credential = null;
				toast.success(m.runtime_credential_name_updated_successfully());
				callback?.();
			})
			.catch(axiosErrorToast);
	}
</script>

<Dialog.Root open={!!credential} {onOpenChange}>
	<Dialog.Content class="max-w-md">
		<Dialog.Header>
			<Dialog.Title>{m.name_runtime_credential()}</Dialog.Title>
			<Dialog.Description>{m.name_runtime_credential_description()}</Dialog.Description>
		</Dialog.Header>
		<form onsubmit={preventDefault(onSubmit)}>
			<div class="grid items-center gap-4 sm:grid-cols-4">
				<Field.Label for="runtime-credential-name" class="sm:text-right">{m.name()}</Field.Label>
				<Input id="runtime-credential-name" maxlength={50} bind:value={name} class="col-span-3" />
			</div>
			<Dialog.Footer class="mt-4">
				<Button type="submit">{m.save()}</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
