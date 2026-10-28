<script lang="ts">
	import Input from '#lib/components/ui/input/input.svelte';
	import * as Popover from '#lib/components/ui/popover/index.js';
	import { cn } from '#lib/utils/style.ts';

	let {
		value = $bindable(''),
		placeholder,
		suggestionLimit = 5,
		suggestions,
		disabled = false
	}: {
		value: string;
		placeholder: string;
		suggestionLimit?: number;
		suggestions: string[];
		disabled?: boolean;
	} = $props();

	let filteredSuggestions: string[] = $state(suggestions.slice(0, suggestionLimit));
	let selectedIndex = $state(-1);

	let isInputFocused = $state(false);

	function handleSuggestionClick(suggestion: (typeof suggestions)[0]) {
		value = suggestion;
		filteredSuggestions = [];
	}

	function handleOnInput() {
		filteredSuggestions = suggestions
			.filter((s) => s.includes(value.toLowerCase()))
			.slice(0, suggestionLimit);
	}

	function handleKeydown(e: KeyboardEvent) {
		if (!isOpen) return;
		switch (e.key) {
			case 'ArrowDown':
				selectedIndex = Math.min(selectedIndex + 1, filteredSuggestions.length - 1);
				break;
			case 'ArrowUp':
				selectedIndex = Math.max(selectedIndex - 1, -1);
				break;
			case 'Enter':
				if (selectedIndex >= 0) {
					handleSuggestionClick(filteredSuggestions[selectedIndex]);
					e.preventDefault();
				}
				break;
			case 'Escape':
				isInputFocused = false;
				break;
		}
	}

	let isOpen = $derived(filteredSuggestions.length > 0 && isInputFocused);

	$effect(() => {
		// Reset selection when suggestions change
		if (filteredSuggestions) {
			selectedIndex = -1;
		}
	});

	$effect(() => handleOnInput());
</script>

<div
	class="grid w-full"
	role="combobox"
	onkeydown={handleKeydown}
	aria-controls="suggestion-list"
	aria-expanded={isOpen}
	tabindex="-1"
>
	<Input
		{placeholder}
		{disabled}
		bind:value
		oninput={handleOnInput}
		onfocus={() => (isInputFocused = true)}
		onblur={() => (isInputFocused = false)}
	/>
	<Popover.Root open={isOpen}>
		<Popover.Trigger tabindex={-1} class="h-0 w-full" aria-hidden />
		<Popover.Content
			class="bg-popover/72 border-fg/6 dark:border-fg/10 gap-0! p-1 backdrop-blur-[40px] backdrop-saturate-150"
			sameWidth
			sideOffset={5}
			trapFocus={false}
			interactOutsideBehavior="ignore"
			onOpenAutoFocus={(e: Event) => e.preventDefault()}
			onCloseAutoFocus={(e: Event) => e.preventDefault()}
			avoidCollisions={false}
			strategy="absolute"
		>
			{#each filteredSuggestions as suggestion, index (suggestion)}
				<div
					role="button"
					tabindex="0"
					onmousedown={() => handleSuggestionClick(suggestion)}
					onkeydown={(e) => {
						if (e.key === 'Enter') handleSuggestionClick(suggestion);
					}}
					class={cn(
						'text-mono hover:bg-fg/6 flex min-h-9 w-full cursor-default items-center rounded-sm px-2.5 font-mono select-none',
						selectedIndex === index && 'bg-fg/6 outline-ring outline-2 -outline-offset-2'
					)}
				>
					{suggestion}
				</div>
			{/each}
		</Popover.Content>
	</Popover.Root>
</div>
