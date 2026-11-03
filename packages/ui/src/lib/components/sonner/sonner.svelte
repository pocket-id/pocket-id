<script lang="ts">
	import { Toaster as Sonner, type ToasterProps as SonnerProps } from 'svelte-sonner';
	import CircleCheckIcon from '~icons/solar/check-circle-bold';
	import OctagonXIcon from '~icons/solar/close-circle-bold';
	import InfoIcon from '~icons/solar/info-circle-bold';
	import TriangleAlertIcon from '~icons/solar/danger-triangle-bold';

	let { ...restProps }: SonnerProps = $props();

	const buttonClass =
		'h-7! rounded-xs! bg-transparent! px-2! text-caption! font-medium! text-inverse-fg! hover:bg-inverse-hover!';
</script>

<!-- svelte-sonner ships unlayered CSS, which beats layered Tailwind utilities, so overrides need the important modifier -->
<Sonner
	class="toaster group z-(--z-toast)! font-sans!"
	style="--normal-bg: var(--inverse); --normal-text: var(--inverse-fg); --normal-border: var(--inverse-border); --border-radius: var(--radius-lg);"
	gap={8}
	toastOptions={{
		classes: {
			toast:
				'w-full! max-w-[calc(100vw-2rem)]! sm:w-(--toast-w)! items-start! gap-2! p-3! text-caption! shadow-overlay! transition-[transform,opacity]! duration-(--dur-base)! ease-out! focus-visible:outline-2! focus-visible:-outline-offset-2! focus-visible:outline-inverse-fg! data-[swiping=true]:transition-none!',
			icon: 'mt-px!',
			description: 'text-micro! font-normal! text-inverse-muted!',
			actionButton: buttonClass,
			cancelButton: buttonClass,
			closeButton:
				'border-inverse-border! bg-inverse! text-inverse-muted! hover:bg-inverse-hover! hover:text-inverse-fg!'
		}
	}}
	{...restProps}
>
	{#snippet loadingIcon()}
		<!-- Solar has no spinner -->
		<svg
			class="text-inverse-muted size-3.5 animate-spin"
			viewBox="0 0 24 24"
			fill="none"
			aria-hidden="true"
		>
			<path
				d="M21 12a9 9 0 1 1-6.22-8.56"
				stroke="currentColor"
				stroke-width="2.5"
				stroke-linecap="round"
			/>
		</svg>
	{/snippet}
	{#snippet successIcon()}
		<CircleCheckIcon class="text-success-solid size-3.5" />
	{/snippet}
	{#snippet errorIcon()}
		<OctagonXIcon class="text-danger-solid size-3.5" />
	{/snippet}
	{#snippet infoIcon()}
		<InfoIcon class="text-inverse-muted size-3.5" />
	{/snippet}
	{#snippet warningIcon()}
		<TriangleAlertIcon class="text-warning-solid size-3.5" />
	{/snippet}
</Sonner>
