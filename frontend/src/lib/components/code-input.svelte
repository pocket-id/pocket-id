<script lang="ts">
	import { onMount } from 'svelte';
	import { m } from '$lib/paraglide/messages';

	let {
		value = $bindable(''),
		length = 8,
		disabled = false,
		autofocus = false,
		onsubmit
	}: {
		value: string;
		length?: number;
		disabled?: boolean;
		autofocus?: boolean;
		onsubmit?: () => void;
	} = $props();

	let inputEl: HTMLInputElement | null = $state(null);
	let focused = $state(false);

	// A single input holds the whole value; the boxes are display-only. Using one input instead of
	// one per character means focus never moves while typing, which is what keeps the on-screen
	// keyboard from dismissing after every character on mobile browsers.
	// dashes added every 4 chars for large codes on large devices, otherwise in middle of short code only.

	const chars = $derived(Array.from({ length }, (_, i) => value[i] ?? ''));
	const activeIndex = $derived(Math.min(value.length, length - 1));

	function handleInput(e: Event) {
		const el = e.target as HTMLInputElement;
		value = el.value.slice(0, length);
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && onsubmit) {
			e.preventDefault();
			onsubmit();
		}
	}

	onMount(() => {
		if (autofocus && !disabled) inputEl?.focus();
	});
</script>

<div
	class={length > 6
		? 'relative max-sm:grid max-sm:[grid-template-columns:repeat(8,40px)] max-sm:justify-center max-sm:justify-items-center max-sm:gap-x-0.5 max-sm:gap-y-0 sm:flex sm:flex-row sm:items-center sm:justify-center sm:gap-x-1.5'
		: 'relative flex flex-nowrap items-center justify-center gap-x-1.5'}
>
	{#each chars as char, i}
		{#if length > 6 && i === 8}
			<div class="col-span-full hidden max-sm:block w-full h-11 px-2 my-0" aria-hidden="true">
				<svg
					class="w-full h-full text-muted-foreground"
					viewBox="0 0 100 20"
					preserveAspectRatio="none"
					fill="none"
					stroke="currentColor"
					stroke-linecap="round"
					stroke-linejoin="round"
				>
					<path
						d="M 98,4 
							C 98,8 97.5,8 96,8 
							L 4,8 
							C 2.5,8 2,8 2,16"
						stroke-width="0.85"
					/>

					<polygon
						points="1,14 2,16 3,14"
						fill="currentColor"
						stroke="currentColor"
						stroke-width="1"
					/>
				</svg>
			</div>
		{/if}
		{#if i > 0}
			{#if length > 6 && i % 4 === 0}
				<span
					class={`text-muted-foreground hidden text-2xl font-light sm:inline ${i === 8 ? 'max-sm:hidden' : ''}`}
					aria-hidden="true">–</span
				>
			{:else if length <= 6 && i === Math.floor(length / 2)}
				<span class="text-muted-foreground text-2xl font-light" aria-hidden="true">–</span>
			{/if}
		{/if}

		<div
			class={`border-input bg-background dark:bg-input/30 flex h-12 w-full max-w-[40px] items-center justify-center rounded-lg border text-center text-lg font-bold shadow-xs transition-all ${!disabled && focused && i === activeIndex ? 'border-ring ring-ring/50 ring-[3px]' : ''} ${disabled ? 'opacity-50' : ''}`}
			aria-hidden="true"
		>
			{char}
		</div>
	{/each}

	{#if !disabled}
		<input
			bind:this={inputEl}
			{value}
			oninput={handleInput}
			name="auth_code"
			data-bwignore="true"
			data-lpignore="true"
			data-1p-ignore="true"
			onkeydown={handleKeydown}
			onfocus={() => (focused = true)}
			onblur={() => (focused = false)}
			inputmode="text"
			autocapitalize="none"
			autocomplete="one-time-code"
			autocorrect="off"
			spellcheck="false"
			maxlength={length}
			aria-label={m.code()}
			class="absolute inset-0 h-full w-full cursor-pointer opacity-0 outline-none"
		/>
	{/if}
</div>
