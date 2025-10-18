<script lang="ts">
	import UrlFileInput from '$lib/components/form/url-file-input.svelte';
	import ImageBox from '$lib/components/image-box.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Label } from '$lib/components/ui/label';
	import { m } from '$lib/paraglide/messages';
	import { LucideX } from '@lucide/svelte';

	let {
		logoDataURL,
		clientName,
		resetLogo,
		onLogoChange,
		label,
		id = 'logo'
	}: {
		logoDataURL: string | null;
		clientName: string;
		resetLogo: () => void;
		onLogoChange: (file: File | string | null) => void;
		label?: string;
		id?: string;
	} = $props();
</script>

<Label for={id}>{label || m.logo()}</Label>
<div class="flex items-end gap-4">
	{#if logoDataURL}
		<div class="flex items-start gap-4">
			<div class="relative shrink-0">
				<ImageBox class="size-24" src={logoDataURL} alt={m.name_logo({ name: clientName })} />
				<Button
					variant="destructive"
					size="icon"
					onclick={resetLogo}
					class="absolute -right-2 -top-2 size-6 rounded-full shadow-md"
				>
					<LucideX class="size-3" />
				</Button>
			</div>
		</div>
	{/if}
	<div class="flex flex-col gap-3">
		<div class="flex flex-wrap items-center gap-2">
			<UrlFileInput {id} label={m.upload_logo()} accept="image/*" onchange={onLogoChange} />
		</div>
	</div>
</div>
