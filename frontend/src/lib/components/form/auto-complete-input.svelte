<script lang="ts">
	import Input from '$lib/components/ui/input/input.svelte';
	import * as Popover from '$lib/components/ui/popover/index.js';

	let {
		value = $bindable(''),
		placeholder,
		suggestionLimit = 5,
		suggestions
	}: {
		value: string;
		placeholder: string;
		suggestionLimit?: number;
		suggestions: string[];
	} = $props();

	let filteredSuggestions: string[] = $state(suggestions.slice(0, suggestionLimit));
	let selectedIndex = $state(-1);
	let keyError: string | undefined = $state();

	let isInputFocused = $state(false);

	function handleSuggestionClick(suggestion: (typeof suggestions)[0]) {
		value = suggestion;
		filteredSuggestions = [];
	}

	function handleOnInput() {
		if (value.length > 0 && !/^[A-Za-z0-9]*$/.test(value)) {
			keyError = 'Only alphanumeric characters are allowed';
			return;
		} else {
			keyError = undefined;
		}

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
		bind:value
		oninput={handleOnInput}
		onfocus={() => (isInputFocused = true)}
		onblur={() => (isInputFocused = false)}
	/>
	{#if keyError}
		<p class="mt-1 text-sm text-red-500">{keyError}</p>
	{/if}
	<Popover.Root
		open={isOpen}
		disableFocusTrap
		openFocus={() => {}}
		closeOnOutsideClick={false}
		closeOnEscape={false}
	>
		<Popover.Trigger tabindex={-1} class="h-0 w-full" aria-hidden />
		<Popover.Content class="p-0" sideOffset={5} sameWidth>
			{#each filteredSuggestions as suggestion, index}
				<div
					role="button"
					tabindex="0"
					onmousedown={() => handleSuggestionClick(suggestion)}
					onkeydown={(e) => {
						if (e.key === 'Enter') handleSuggestionClick(suggestion);
					}}
					class="relative flex w-full cursor-default select-none items-center rounded-sm py-1.5 pl-8 pr-2 text-sm outline-none hover:bg-accent hover:text-accent-foreground data-[disabled]:pointer-events-none data-[disabled]:opacity-50 {selectedIndex ===
					index
						? 'bg-accent text-accent-foreground'
						: ''}"
				>
					{suggestion}
				</div>
			{/each}
		</Popover.Content>
	</Popover.Root>
</div>
