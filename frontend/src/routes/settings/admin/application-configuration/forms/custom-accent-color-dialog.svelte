<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label/index.js';
	import { m } from '$lib/paraglide/messages';
	import { preventDefault } from '$lib/utils/event-util';

	let {
		open = $bindable(false),
		onApply
	}: {
		open: boolean;
		onApply: (color: string) => void;
	} = $props();

	let customColorInput = $state('');

	function applyCustomColor() {
		if (!isValidColor(customColorInput)) return;
		onApply(customColorInput);
		open = false;
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
		}
		open = newOpen;
	}
</script>

<Dialog.Root {open} {onOpenChange}>
	<Dialog.Content class="max-w-md">
		<Dialog.Header>
			<Dialog.Title class="flex items-center gap-2">{m.custom_accent_color()}</Dialog.Title>
			<Dialog.Description>
				{m.custom_accent_color_description()}
			</Dialog.Description>
		</Dialog.Header>

		<form onsubmit={preventDefault(applyCustomColor)}>
			<div class="space-y-4">
				<div>
					<Label for="custom-color-input" class="text-sm font-medium">{m.color_value()}</Label>
					<div class="flex items-center gap-2">
						<div class="w-full transition">
							<Input
								id="custom-color-input"
								bind:value={customColorInput}
								placeholder="#3b82f6"
								class="mt-1 flex-1"
							/>
						</div>
						<div
							class={{
								'border-border mt-1 rounded-lg border-1 transition-all duration-200 ease-in-out': true,
								'h-9 w-9': isValidColor(customColorInput),
								'h-0 w-0': !isValidColor(customColorInput)
							}}
							style="background-color: {customColorInput}"
						></div>
					</div>
				</div>
			</div>

			<Dialog.Footer class="mt-6">
				<Button variant="secondary" onclick={() => onOpenChange(false)}>{m.cancel()}</Button>
				<Button type="submit" disabled={!customColorInput || !isValidColor(customColorInput)}
					>{m.apply()}</Button
				>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
