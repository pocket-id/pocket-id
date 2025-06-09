<script lang="ts">
	import { Label } from '$lib/components/ui/label/index.js';
	import * as RadioGroup from '$lib/components/ui/radio-group/index.js';
	import UserService from '$lib/services/user-service';
	import userStore from '$lib/stores/user-store';
	import { toast } from 'svelte-sonner';
	import { Check } from '@lucide/svelte';
	import { applyAccentColor } from '$lib/utils/accent-color-util';

	const userService = new UserService();

	const accentColors = [
		{ value: 'default', label: 'Default', color: 'hsl(var(--primary))' },
		{ value: 'red', label: 'Red', color: 'oklch(0.637 0.237 25.331)' },
		{ value: 'rose', label: 'Rose', color: 'oklch(0.658 0.218 12.180)' },
		{ value: 'orange', label: 'Orange', color: 'oklch(0.705 0.213 47.604)' },
		{ value: 'green', label: 'Green', color: 'oklch(0.723 0.219 149.579)' },
		{ value: 'blue', label: 'Blue', color: 'oklch(0.623 0.214 259.815)' },
		{ value: 'yellow', label: 'Yellow', color: 'oklch(0.795 0.184 86.047)' },
		{ value: 'violet', label: 'Violet', color: 'oklch(0.649 0.221 285.75)' }
	];

	async function updateAccentColor(accentValue: string) {
		applyAccentColor(accentValue);

		try {
			await userService.updateCurrent({
				...$userStore!,
				accentColor: accentValue
			});

			userStore.setUser({
				...$userStore!,
				accentColor: accentValue
			});

			toast.success('Accent color updated successfully');
		} catch (error) {
			toast.error('Failed to update accent color');
		}
	}

	let selectedAccent = $derived($userStore?.accentColor || 'default');
</script>

<RadioGroup.Root
	class="flex flex-wrap gap-2"
	value={selectedAccent}
	onValueChange={updateAccentColor}
>
	{#each accentColors as accent}
		<div class="relative">
			<RadioGroup.Item value={accent.value} id={accent.value} class="sr-only" />
			<Label for={accent.value} class="group cursor-pointer">
				<div
					class="relative h-8 w-8 rounded-full border-2 transition-all duration-200 {selectedAccent ===
					accent.value
						? 'border-primary ring-primary ring-2 ring-offset-2'
						: 'group-hover:border-primary group-hover:ring-primary border-gray-200 group-hover:ring-1 group-hover:ring-offset-1'}"
					style="background-color: {accent.color}"
				>
					{#if selectedAccent === accent.value}
						<div class="absolute inset-0 flex items-center justify-center">
							<Check class="size-4 text-white drop-shadow-sm" />
						</div>
					{/if}
				</div>
			</Label>
		</div>
	{/each}
</RadioGroup.Root>
