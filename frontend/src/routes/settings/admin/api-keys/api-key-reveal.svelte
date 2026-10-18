<script lang="ts">
	import CopyField from '#lib/components/copy-field.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import { getLocale } from '#lib/paraglide/runtime.js';
	import type { ApiKeyResponse } from '#lib/types/api-key.type.ts';
	import { LucideInfo } from '@lucide/svelte';

	let { response }: { response: ApiKeyResponse } = $props();
</script>

<dl class="text-control grid grid-cols-[auto_1fr] gap-x-6 gap-y-2">
	<dt class="text-fg-muted">{m.name()}</dt>
	<dd class="wrap-anywhere">{response.apiKey.name}</dd>
	{#if response.apiKey.description}
		<dt class="text-fg-muted">{m.description()}</dt>
		<dd class="wrap-anywhere">{response.apiKey.description}</dd>
	{/if}
	<dt class="text-fg-muted">{m.expires()}</dt>
	<dd>
		{new Date(response.apiKey.expiresAt).toLocaleDateString(getLocale(), { dateStyle: 'medium' })}
	</dd>
</dl>
<p class="text-caption text-fg-muted flex items-start gap-1.5">
	<LucideInfo class="mt-[3px] size-3 shrink-0" />
	{m.copy_key_now_before_closing()}
</p>
<CopyField label={m.api_key()} value={response.token} copyLabel={m.copy_api_key()} />
