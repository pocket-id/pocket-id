<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog';
	import Input from '$lib/components/ui/input/input.svelte';
	import Label from '$lib/components/ui/label/label.svelte';
	import { m } from '$lib/paraglide/messages';

	let {
		oneTimeLink = $bindable()
	}: {
		oneTimeLink: string | null;
	} = $props();

	function onOpenChange(open: boolean) {
		if (!open) {
			oneTimeLink = null;
		}
	}
</script>

<Dialog.Root open={!!oneTimeLink} {onOpenChange}>
	<Dialog.Content class="max-w-md">
		<Dialog.Header>
			<Dialog.Title>{m.one_time_link()}</Dialog.Title>
			<Dialog.Description>{m.use_this_link_to_sign_in_once()}</Dialog.Description>
		</Dialog.Header>
		<Label for="one-time-link">{m.one_time_link()}</Label>
		<Input id="one-time-link" value={oneTimeLink} readonly />
	</Dialog.Content>
</Dialog.Root>
