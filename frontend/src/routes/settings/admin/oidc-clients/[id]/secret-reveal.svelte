<script lang="ts">
	import CopyField from '#lib/components/copy-field.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import { LucideInfo } from '@lucide/svelte';

	let { label, value, testId }: { label: string; value: string; testId: string } = $props();
</script>

<div class="bg-surface flex flex-col gap-2 rounded-md border p-4">
	<p class="text-control text-fg-muted flex items-start gap-1.5">
		<LucideInfo class="mt-[3px] size-3.5 shrink-0" />
		{m.copy_it_now_you_wont_see_it_again()}
	</p>
	<CopyField {label} {value} {testId} copyLabel={m.copy_label({ label: label.toLowerCase() })} />
</div>
