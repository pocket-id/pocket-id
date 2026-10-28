<script lang="ts">
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Tooltip from '#lib/components/ui/tooltip/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { LucideRotateCcw } from '@lucide/svelte';
	import { tick } from 'svelte';

	let {
		label,
		onReset
	}: {
		label: string;
		onReset: () => void;
	} = $props();

	let root = $state<HTMLElement>();

	// The reset button leaves with the changed state, so focus moves to the field it reset
	async function reset() {
		const field = root?.closest(':has(input)');
		onReset();
		await tick();
		field?.querySelector('input')?.focus();
	}
</script>

<!-- The field-level form of the setting row's reset, for fields whose label lives in a shared input component -->
<span bind:this={root} class="absolute top-0 right-0">
	<Tooltip.Provider>
		<Tooltip.Root>
			<Tooltip.Trigger>
				{#snippet child({ props })}
					<Button
						{...props}
						variant="ghost"
						size="icon-xs"
						aria-label={m.reset_setting_to_default({ setting: label })}
						onclick={reset}
					>
						<LucideRotateCcw />
					</Button>
				{/snippet}
			</Tooltip.Trigger>
			<Tooltip.Content>{m.reset_to_default()}</Tooltip.Content>
		</Tooltip.Root>
	</Tooltip.Provider>
</span>
