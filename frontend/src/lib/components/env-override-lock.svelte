<script lang="ts">
	import * as Tooltip from '@pocket-id/ui/tooltip';
	import { m } from '#lib/paraglide/messages.js';
	import LockIcon from '~icons/solar/lock-keyhole-minimalistic-bold-duotone';
</script>

<!-- A span instead of a button, because the disabled fieldset around a locked setting would disable a button and with it the tooltip -->
<!-- It still takes focus, so keyboard users can reveal the tooltip too -->
<Tooltip.Root>
	<Tooltip.Trigger>
		{#snippet child({ props })}
			<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
			<span
				{...props}
				role="img"
				tabindex="0"
				aria-label={m.config_set_by_env()}
				class="text-fg-muted hover:text-fg focus-visible:outline-ring inline-grid size-5 shrink-0 place-items-center rounded-sm transition-colors duration-(--dur-fast) focus-visible:outline-2"
			>
				<LockIcon class="size-3.5" />
			</span>
		{/snippet}
	</Tooltip.Trigger>
	<Tooltip.Content>{m.config_set_by_env()}</Tooltip.Content>
</Tooltip.Root>
