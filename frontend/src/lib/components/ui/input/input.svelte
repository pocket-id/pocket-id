<script lang="ts" module>
	// Shared by inputs, textareas and select triggers
	// The dark: classes repeat the focus and invalid borders because the dark variant sorts after every state variant
	export const filledFieldClass =
		'border border-transparent bg-fill-muted text-fg outline-hidden hover:bg-fill-hover focus-visible:border-ring focus-visible:bg-surface focus-visible:ring-(length:--halo-width) focus-visible:ring-ring-halo aria-invalid:border-danger-solid aria-invalid:ring-danger-solid/20 disabled:cursor-not-allowed disabled:opacity-45 dark:border-fg/10 dark:bg-fg/6 dark:hover:bg-fg/9 dark:focus-visible:border-ring dark:focus-visible:bg-fg/7 dark:aria-invalid:border-danger-solid';

	// Read-only fields drop the fill for a hairline so they never look editable
	export const readOnlyFieldClass =
		'read-only:border-border read-only:bg-transparent! dark:read-only:border-border';
</script>

<script lang="ts">
	import type { HTMLInputAttributes, HTMLInputTypeAttribute } from 'svelte/elements';
	import { cn, type WithElementRef } from '#lib/utils/style.js';

	type InputType = Exclude<HTMLInputTypeAttribute, 'file'>;

	type Props = WithElementRef<
		Omit<HTMLInputAttributes, 'type'> &
			({ type: 'file'; files?: FileList } | { type?: InputType; files?: undefined })
	>;

	let {
		ref = $bindable(null),
		value = $bindable(),
		type,
		files = $bindable(),
		class: className,
		'data-slot': dataSlot = 'input',
		...restProps
	}: Props = $props();

	// Phone inputs use 16px text so iOS does not zoom on focus, and mono values elsewhere use the mono size because mono glyphs run larger than sans ones
	const inputClass = $derived(
		cn(
			filledFieldClass,
			readOnlyFieldClass,
			'h-(--h-md) w-full min-w-0 rounded-sm px-3.5 text-body placeholder:text-fg-muted autofill:shadow-[inset_0_0_0_100px_var(--fill-muted)] autofill:[-webkit-text-fill-color:var(--fg)] max-sm:text-[16px] sm:[&.font-mono]:text-mono file:inline-flex file:h-full file:border-0 file:bg-transparent file:font-medium file:text-fg',
			className
		)
	);
</script>

{#if type === 'file'}
	<input
		bind:this={ref}
		data-slot={dataSlot}
		class={inputClass}
		type="file"
		bind:files
		bind:value
		{...restProps}
	/>
{:else}
	<input bind:this={ref} data-slot={dataSlot} class={inputClass} {type} bind:value {...restProps} />
{/if}
