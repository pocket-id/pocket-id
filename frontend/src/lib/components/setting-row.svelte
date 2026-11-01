<script lang="ts">
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Tooltip from '#lib/components/ui/tooltip/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { cn } from '#lib/utils/style.ts';
	import { LucideInfo, LucideRotateCcw } from '@lucide/svelte';
	import { tick, type Component, type Snippet } from 'svelte';

	type ControlProps = { id: string; 'aria-describedby': string | undefined };

	let {
		title,
		description,
		icon: Icon,
		stacked = false,
		disabledReason,
		consequence,
		changed = false,
		onReset,
		control
	}: {
		title: string;
		description?: string | Snippet;
		icon?: Component;
		stacked?: boolean;
		// Dims a dependent row; the reason ("Requires Allow sign-ups") is announced with the control, which the caller disables
		disabledReason?: string;
		consequence?: string;
		changed?: boolean;
		onReset?: () => void;
		// Spread the props on the control so the title labels it; leave `id` off for action buttons, whose own text is their name
		control: Snippet<[ControlProps]>;
	} = $props();

	const uid = $props.id();
	const controlId = `${uid}-control`;
	let controlContainer = $state<HTMLElement>();

	const describedBy = $derived(
		[
			description && `${uid}-desc`,
			consequence && `${uid}-consequence`,
			disabledReason && `${uid}-reason`,
			changed && `${uid}-changed`
		]
			.filter(Boolean)
			.join(' ') || undefined
	);

	async function reset() {
		onReset?.();
		// The reset button leaves with the changed state, so focus moves to the control it reset
		await tick();
		controlContainer?.querySelector<HTMLElement>('button, input, textarea, [tabindex]')?.focus();
	}
</script>

<div
	class={cn(
		'flex min-h-[68px] gap-4 py-4 shadow-[inset_0_-1px_0_var(--border)] transition-opacity duration-(--dur-base) ease-out last:shadow-none',
		stacked ? 'items-start' : 'items-center max-sm:items-start',
		disabledReason && 'opacity-45'
	)}
>
	{#if Icon}
		<span
			aria-hidden="true"
			class="bg-surface text-fg grid size-9 shrink-0 place-items-center rounded-full shadow-[inset_0_0_0_1px_var(--border)]"
		>
			<Icon class="size-[18px]" />
		</span>
	{/if}
	<div class="flex min-w-0 flex-1 flex-wrap items-center gap-x-4 gap-y-3 max-sm:items-start">
		<div class="flex min-w-0 flex-1 flex-col gap-0.5">
			<label for={controlId} class="text-heading text-fg">
				{title}
				{#if changed}
					<span class="bg-fg ms-1.5 inline-block size-1.5 rounded-full align-middle"></span>
					<span id="{uid}-changed" class="sr-only">{m.changed_from_the_default()}</span>
				{/if}
			</label>
			{#if description}
				<p id="{uid}-desc" class="text-caption text-fg-muted max-w-[68ch]">
					{#if typeof description === 'string'}{description}{:else}{@render description()}{/if}
				</p>
			{/if}
			{#if consequence}
				<p id="{uid}-consequence" class="text-caption text-fg-muted mt-1 flex items-start gap-1.5">
					<LucideInfo class="mt-[3px] size-3 shrink-0" />
					{consequence}
				</p>
			{/if}
			{#if disabledReason}
				<span id="{uid}-reason" class="sr-only">{disabledReason}</span>
			{/if}
		</div>
		<!-- Outside the control's box, so a control that wraps under the text leaves the reset on the title line -->
		{#if changed && onReset}
			<Tooltip.Root>
				<Tooltip.Trigger>
					{#snippet child({ props })}
						<Button
							{...props}
							variant="ghost"
							size="icon-xs"
							class={cn(!stacked && 'sm:-me-3')}
							aria-label={m.reset_setting_to_default({ setting: title })}
							onclick={reset}
						>
							<LucideRotateCcw />
						</Button>
					{/snippet}
				</Tooltip.Trigger>
				<Tooltip.Content>{m.reset_to_default()}</Tooltip.Content>
			</Tooltip.Root>
		{/if}
		<div
			bind:this={controlContainer}
			class={cn(
				'flex items-center gap-2',
				stacked
					? 'basis-full'
					: 'shrink-0 justify-end max-sm:not-has-[>[data-slot=switch]:only-child]:basis-full max-sm:not-has-[>[data-slot=switch]:only-child]:justify-start'
			)}
		>
			{@render control({ id: controlId, 'aria-describedby': describedBy })}
		</div>
	</div>
</div>
