<script lang="ts">
	import { Button } from '@pocket-id/ui/button';
	import * as Tooltip from '@pocket-id/ui/tooltip';
	import { m } from '#lib/paraglide/messages.js';
	import { cn } from '@pocket-id/ui/utils';
	import CheckIcon from '~icons/solar/check-linear';
	import CopyIcon from '~icons/solar/copy-bold';

	let {
		value,
		label = m.copy(),
		labeled = false,
		class: className
	}: {
		value: string;
		// The accessible name, such as "Copy client ID", which stays the same while the button shows "Copied"
		label?: string;
		// A secondary button that reads "Copy" instead of an icon button with a tooltip
		labeled?: boolean;
		class?: string;
	} = $props();

	let copied = $state(false);
	let revertTimer: ReturnType<typeof setTimeout> | undefined;

	async function copy() {
		await navigator.clipboard.writeText(value);
		copied = true;
		clearTimeout(revertTimer);
		revertTimer = setTimeout(() => (copied = false), 1500);
	}

	const markClass =
		'col-start-1 row-start-1 transition-[opacity,scale] duration-(--dur-fast) ease-out';
</script>

{#snippet icon()}
	<span class="grid" aria-hidden="true">
		<CopyIcon class={cn(markClass, copied && 'scale-60 opacity-0')} />
		<CheckIcon
			class={cn(markClass, !copied && 'scale-60 opacity-0', copied && 'duration-(--dur-base)')}
		/>
	</span>
{/snippet}

{#if labeled}
	<!-- The label cell holds both words, so the button keeps its width when it reads "Copied" -->
	<Button
		variant="secondary"
		size="sm"
		class={className}
		aria-label={label === m.copy() ? undefined : label}
		onclick={copy}
	>
		{@render icon()}
		<span class="grid">
			<span class={cn('col-start-1 row-start-1 transition-opacity', copied && 'opacity-0')}>
				{m.copy()}
			</span>
			<span
				class={cn('col-start-1 row-start-1 transition-opacity', !copied && 'opacity-0')}
				aria-hidden="true"
			>
				{m.copied()}
			</span>
		</span>
	</Button>
{:else}
	<Tooltip.Provider>
		<Tooltip.Root disableCloseOnTriggerClick>
			<Tooltip.Trigger>
				{#snippet child({ props })}
					<Button
						{...props}
						variant="ghost"
						size="icon-xs"
						class={className}
						aria-label={label}
						onclick={copy}
					>
						{@render icon()}
					</Button>
				{/snippet}
			</Tooltip.Trigger>
			<Tooltip.Content>{copied ? m.copied() : label}</Tooltip.Content>
		</Tooltip.Root>
	</Tooltip.Provider>
{/if}
<span class="sr-only" aria-live="polite">{copied ? m.copied() : ''}</span>
