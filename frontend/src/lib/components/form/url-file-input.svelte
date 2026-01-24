<script lang="ts">
	import FileInput from '$lib/components/form/file-input.svelte';
	import FormattedMessage from '$lib/components/formatted-message.svelte';
	import { Button, buttonVariants } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import * as Popover from '$lib/components/ui/popover';
	import { m } from '$lib/paraglide/messages';
	import { cn } from '$lib/utils/style';
	import { LucideChevronDown } from '@lucide/svelte';

	let {
		label,
		accept,
		onchange,
		id = 'file-input'
	}: {
		label: string;
		accept?: string;
		onchange: (file: File | string | null) => void;
		id?: string;
	} = $props();

	let url = $state('');
	let hasError = $state(false);

	async function handleFileChange(e: Event) {
		const file = (e.target as HTMLInputElement).files?.[0] || null;
		url = '';
		hasError = false;
		onchange(file);
	}

	async function handleUrlChange(e: Event) {
		const url = (e.target as HTMLInputElement).value.trim();
		if (!url) return;

		try {
			new URL(url);
			hasError = false;
		} catch {
			hasError = true;
			return;
		}

		onchange(url);
	}
</script>

<div class="flex">
	<FileInput
		{id}
		variant="secondary"
		{accept}
		onchange={handleFileChange}
		onclick={(e: any) => (e.target.value = '')}
	>
		<Button variant="secondary" class="rounded-r-none">
			{label}
		</Button>
	</FileInput>
	<Popover.Root>
		<Popover.Trigger
			class={cn(buttonVariants({ variant: 'secondary' }), 'rounded-l-none border-l')}
		>
			<LucideChevronDown class="size-4" /></Popover.Trigger
		>
		<Popover.Content class="w-80">
			<Label for="{id}-url" class="text-xs">URL</Label>
			<Input
				id="{id}-url"
				placeholder=""
				value={url}
				oninput={(e) => (url = e.currentTarget.value)}
				onfocusout={handleUrlChange}
				aria-invalid={hasError}
				type="url"
			/>
			{#if hasError}
				<p class="text-destructive mt-1 text-start text-xs">{m.invalid_url()}</p>
			{/if}

			<p class="text-muted-foreground mt-2 text-xs">
				<FormattedMessage m={m.logo_from_url_description()} />
			</p>
		</Popover.Content>
	</Popover.Root>
</div>
