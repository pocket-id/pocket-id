<script lang="ts">
	import { onMount, untrack } from 'svelte';

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

	let chars: string[] = $state(Array.from({ length }, () => ''));
	let inputs: HTMLInputElement[] = $state([]);

	$effect(() => {
		const clean = value.replace(/[^a-zA-Z0-9]/g, '').toUpperCase();
		untrack(() => {
			for (let i = 0; i < length; i++) {
				chars[i] = clean[i] || '';
			}
		});
	});

	function syncValue() {
		value = chars.join('');
	}

	function handleInput(index: number, e: Event) {
		const input = e.target as HTMLInputElement;
		const val = input.value.replace(/[^a-zA-Z0-9]/g, '').toUpperCase();

		if (val.length > 1) {
			for (let i = 0; i < val.length && index + i < length; i++) {
				chars[index + i] = val[i];
			}
			const nextIndex = Math.min(index + val.length, length - 1);
			inputs[nextIndex]?.focus();
		} else {
			chars[index] = val;
			if (val && index < length - 1) {
				inputs[index + 1]?.focus();
			}
		}
		syncValue();
	}

	onMount(() => {
		if (autofocus && !disabled) {
			inputs[0]?.focus();
		}
	});

	function handleKeydown(index: number, e: KeyboardEvent) {
		if (e.key === 'Enter' && onsubmit) {
			e.preventDefault();
			onsubmit();
			return;
		}
		if (e.key === 'Backspace' && !chars[index] && index > 0) {
			chars[index - 1] = '';
			inputs[index - 1]?.focus();
			syncValue();
			e.preventDefault();
		} else if (e.key === 'ArrowLeft' && index > 0) {
			inputs[index - 1]?.focus();
			e.preventDefault();
		} else if (e.key === 'ArrowRight' && index < length - 1) {
			inputs[index + 1]?.focus();
			e.preventDefault();
		}
	}

	function handlePaste(e: ClipboardEvent) {
		e.preventDefault();
		const pasted = (e.clipboardData?.getData('text') || '')
			.replace(/[^a-zA-Z0-9]/g, '')
			.toUpperCase();
		for (let i = 0; i < pasted.length && i < length; i++) {
			chars[i] = pasted[i];
		}
		const nextIndex = Math.min(pasted.length, length - 1);
		inputs[nextIndex]?.focus();
		syncValue();
	}

	function handleFocus(e: FocusEvent) {
		(e.target as HTMLInputElement).select();
	}
</script>

<div class="flex items-center gap-1.5">
	{#each { length } as _, i}
		{#if i === length / 2}
			<span class="text-muted-foreground text-2xl font-light">–</span>
		{/if}
		<input
			bind:this={inputs[i]}
			value={chars[i]}
			oninput={(e) => handleInput(i, e)}
			onkeydown={(e) => handleKeydown(i, e)}
			onpaste={handlePaste}
			onfocus={handleFocus}
			{disabled}
			maxlength={1}
			autocomplete="off"
			class="border-input bg-background dark:bg-input/30 ring-offset-background placeholder:text-muted-foreground
				flex h-12 w-10 items-center justify-center rounded-lg border text-center text-lg font-bold uppercase
				shadow-xs outline-none transition-all
				focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px]
				disabled:pointer-events-none disabled:opacity-50"
		/>
	{/each}
</div>
