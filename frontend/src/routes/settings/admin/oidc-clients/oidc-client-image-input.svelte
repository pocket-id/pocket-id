<script lang="ts">
	import UrlFileInput from '$lib/components/form/url-file-input.svelte';
	import ImageBox from '$lib/components/image-box.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Field from '$lib/components/ui/field';
	import { m } from '$lib/paraglide/messages';
	import { LucideX } from '@lucide/svelte';
	import type { Snippet } from 'svelte';

	let {
		logoDataURL,
		clientName,
		resetLogo,
		onLogoChange,
		light,
		tabTriggers
	}: {
		logoDataURL: string | null;
		clientName: string;
		resetLogo: () => void;
		onLogoChange: (file: File | string | null) => void;
		tabTriggers?: Snippet;
		light: boolean;
	} = $props();

	let id = `oidc-client-logo-${light ? 'light' : 'dark'}`;
</script>

<Field.Label for={id}>{m.logo()}</Field.Label>
<div class="flex h-24 items-end gap-4">
	<div class="flex flex-col gap-2">
		{#if tabTriggers}
			{@render tabTriggers()}
		{/if}
		<div class="flex flex-wrap items-center gap-2">
			<UrlFileInput {id} label={m.upload_logo()} accept="image/*" onchange={onLogoChange} />
		</div>
	</div>
	{#if logoDataURL}
		<div class="flex items-start gap-4">
			<div class="relative shrink-0">
				<ImageBox
					class="size-24 {light ? 'bg-[#F5F5F5]' : 'bg-[#262626]'}"
					src={logoDataURL}
					alt={m.name_logo({ name: clientName })}
				/>
				<Button
					size="icon"
					onclick={resetLogo}
					class="absolute -top-2 -right-2 size-6 rounded-full shadow-md "
				>
					<LucideX class="size-3" />
				</Button>
			</div>
		</div>
	{/if}
</div>
