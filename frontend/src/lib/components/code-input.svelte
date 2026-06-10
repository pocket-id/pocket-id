<script lang="ts">
	import { onMount } from 'svelte';

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
	const normalized = $derived(
		value
			.replace(/[^a-zA-Z0-9]/g, '')
			.toUpperCase()
			.slice(0, length)
	);
	const chars = $derived(Array.from({ length }, (_, i) => normalized[i] ?? ''));
	const activeIndex = $derived(Math.min(normalized.length, length - 1));

	function handleInput(e: Event) {
		const el = e.target as HTMLInputElement;
		value = el.value
			.replace(/[^a-zA-Z0-9]/g, '')
			.toUpperCase()
			.slice(0, length);
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

<div class="relative flex items-center gap-1.5">
	{#each chars as char, i}
		{#if i === Math.floor(length / 2) && length > 1}
			<span class="text-muted-foreground text-2xl font-light" aria-hidden="true">–</span>
		{/if}
		<div
			class="border-input bg-background dark:bg-input/30 flex h-12 w-10 items-center justify-center
				rounded-lg border text-center text-lg font-bold uppercase shadow-xs transition-all
				{!disabled && focused && i === activeIndex ? 'border-ring ring-ring/50 ring-[3px]' : ''}
				{disabled ? 'opacity-50' : ''}"
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
			onkeydown={handleKeydown}
			onfocus={() => (focused = true)}
			onblur={() => (focused = false)}
			inputmode="text"
			autocapitalize="characters"
			autocomplete="off"
			autocorrect="off"
			spellcheck="false"
			maxlength={length}
			aria-label="Verification code"
			class="absolute inset-0 h-full w-full cursor-pointer opacity-0 outline-none"
		/>
	{/if}
</div>
