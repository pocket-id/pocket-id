<script lang="ts">
	import FileInput from '$lib/components/form/file-input.svelte';
	import { Label } from '$lib/components/ui/label';
	import { cn } from '$lib/utils/style';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		id,
		imageClass,
		label,
		image = $bindable(),
		imageURL,
		accept = 'image/png, image/jpeg, image/svg+xml',
		forceColorScheme,
		...restProps
	}: HTMLAttributes<HTMLDivElement> & {
		id: string;
		imageClass: string;
		label: string;
		image: File | null;
		imageURL: string;
		forceColorScheme?: 'light' | 'dark';
		accept?: string;
	} = $props();

	let imageDataURL = $state(imageURL);

	function onImageChange(e: Event) {
		const file = (e.target as HTMLInputElement).files?.[0] || null;
		if (!file) return;

		image = file;

		const reader = new FileReader();
		reader.onload = (event) => {
			imageDataURL = event.target?.result as string;
		};
		reader.readAsDataURL(file);
	}
</script>

<div class="flex flex-col items-start md:flex-row md:items-center" {...restProps}>
	<Label class="w-52" for={id}>{label}</Label>
	<FileInput {id} variant="secondary" {accept} onchange={onImageChange}>
		<div
			class="{forceColorScheme === 'light'
				? 'bg-[#F1F1F5]'
				: forceColorScheme === 'dark'
					? 'bg-[#27272A]'
					: 'bg-muted'} group relative flex items-center rounded"
		>
			<img
				class={cn(
					'h-full w-full rounded object-cover p-3 transition-opacity duration-200 group-hover:opacity-10',
					imageClass
				)}
				src={imageDataURL}
				alt={label}
			/>
			<span
				class="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 transform font-medium opacity-0 transition-opacity group-hover:opacity-100"
			>
				Update
			</span>
		</div>
	</FileInput>
</div>
