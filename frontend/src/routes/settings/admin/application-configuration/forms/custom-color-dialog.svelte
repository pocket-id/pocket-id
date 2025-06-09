<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { Label } from '$lib/components/ui/label/index.js';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Input } from '$lib/components/ui/input';
	import { toast } from 'svelte-sonner';
	import { Palette } from '@lucide/svelte';

	let {
		open = $bindable(false),
		onApply
	}: {
		open: boolean;
		onApply: (color: string) => void;
	} = $props();

	let customColorInput = $state('');
	let customColorPreview = $state('');

	function handleCustomColorInput() {
		// Update preview as user types
		if (isValidColor(customColorInput)) {
			customColorPreview = customColorInput;
		}
	}

	function applyCustomColor() {
		if (isValidColor(customColorInput)) {
			onApply(customColorInput);
			open = false;
			toast.success('Custom accent color applied!');
		} else {
			toast.error('Please enter a valid color (hex, hsl, or oklch)');
		}
	}

	function isValidColor(color: string): boolean {
		// Create a temporary element to test if the color is valid
		const testElement = document.createElement('div');
		testElement.style.color = color;
		return testElement.style.color !== '';
	}

	function onOpenChange(newOpen: boolean) {
		if (!newOpen) {
			customColorInput = '';
			customColorPreview = '';
		}
		open = newOpen;
	}
</script>

<Dialog.Root {open} {onOpenChange}>
	<Dialog.Content class="max-w-md">
		<Dialog.Header>
			<Dialog.Title class="flex items-center gap-2">
				<Palette class="size-5" />
				Custom Accent Color
			</Dialog.Title>
			<Dialog.Description>
				Enter a custom color using valid formats:
				<code class="bg-muted rounded px-1 py-0.5 text-xs">hex</code>,
				<code class="bg-muted rounded px-1 py-0.5 text-xs">hsl()</code>, or
				<code class="bg-muted rounded px-1 py-0.5 text-xs">oklch()</code>
			</Dialog.Description>
		</Dialog.Header>

		<div class="space-y-4">
			<div>
				<Label for="custom-color-input" class="text-sm font-medium">Color Value</Label>
				<Input
					id="custom-color-input"
					bind:value={customColorInput}
					oninput={handleCustomColorInput}
					placeholder="e.g., #3b82f6, hsl(217, 91%, 60%), oklch(0.623 0.214 259.815)"
					class="mt-1"
				/>
			</div>

			{#if customColorPreview && isValidColor(customColorPreview)}
				<div class="space-y-2">
					<Label class="text-sm font-medium">Preview</Label>
					<div class="flex items-center gap-3">
						<div
							class="h-12 w-12 rounded-lg border-2 border-gray-200"
							style="background-color: {customColorPreview}"
						></div>
						<div class="text-muted-foreground font-mono text-sm">
							{customColorPreview}
						</div>
					</div>
				</div>
			{/if}
		</div>

		<Dialog.Footer class="mt-6">
			<Button variant="outline" onclick={() => onOpenChange(false)}>Cancel</Button>
			<Button
				onclick={applyCustomColor}
				disabled={!customColorInput || !isValidColor(customColorInput)}
			>
				Apply Color
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
