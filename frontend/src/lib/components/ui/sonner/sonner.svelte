<script lang="ts">
	import { Toaster as Sonner, type ToasterProps as SonnerProps } from 'svelte-sonner';
	import { mode } from 'mode-watcher';
	import Loader2Icon from '@lucide/svelte/icons/loader-2';
	import CircleCheckIcon from '@lucide/svelte/icons/circle-check';
	import OctagonXIcon from '@lucide/svelte/icons/octagon-x';
	import InfoIcon from '@lucide/svelte/icons/info';
	import TriangleAlertIcon from '@lucide/svelte/icons/triangle-alert';

	let { ...restProps }: SonnerProps = $props();

	const buttonClass =
		'h-7! rounded-xs! bg-transparent! px-2! text-caption! font-medium! text-inverse-fg! hover:bg-inverse-hover!';
</script>

<!-- svelte-sonner ships unlayered CSS, which beats layered Tailwind utilities, so overrides need the important modifier -->
<Sonner
	theme={mode.current}
	class="toaster group z-(--z-toast)! font-sans!"
	style="--normal-bg: var(--inverse); --normal-text: var(--inverse-fg); --normal-border: var(--inverse-border); --border-radius: var(--radius-lg);"
	gap={8}
	mobileOffset={{ bottom: 'calc(76px + env(safe-area-inset-bottom))' }}
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
		<Loader2Icon class="text-inverse-muted size-3.5 animate-spin" />
	{/snippet}
	{#snippet successIcon()}
		<CircleCheckIcon class="fill-success-solid text-inverse size-3.5" />
	{/snippet}
	{#snippet errorIcon()}
		<OctagonXIcon class="fill-danger-solid text-inverse size-3.5" />
	{/snippet}
	{#snippet infoIcon()}
		<InfoIcon class="fill-inverse-muted text-inverse size-3.5" />
	{/snippet}
	{#snippet warningIcon()}
		<TriangleAlertIcon class="fill-warning-solid text-inverse size-3.5" />
	{/snippet}
</Sonner>
