<script lang="ts">
	import FileInput from '$lib/components/form/file-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Label } from '$lib/components/ui/label';
	import { cn } from '$lib/utils/style';
	import { LucideImageOff, LucideUpload, LucideX } from '@lucide/svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		id,
		imageClass,
		label,
		image = $bindable(),
		imageURL,
		accept = 'image/png, image/jpeg, image/svg+xml, image/gif, image/webp, image/avif, image/heic',
		forceColorScheme,
		isResetable = false,
		isImageSet = $bindable(true),
		...restProps
	}: HTMLAttributes<HTMLDivElement> & {
		id: string;
		imageClass: string;
		label: string;
		image: File | null | undefined;
		imageURL: string;
		forceColorScheme?: 'light' | 'dark';
		accept?: string;
		isResetable?: boolean;
		isImageSet?: boolean;
	} = $props();

	let imageDataURL = $state(imageURL);

	$effect(() => {
		if (image) {
			isImageSet = true;
		}
	});

	function onImageChange(e: Event) {
		const file = (e.target as HTMLInputElement).files?.[0] || undefined;
		if (!file) return;

		image = file;
		imageDataURL = URL.createObjectURL(file);
	}

	function onReset() {
		image = null;
		imageDataURL = imageURL;
		isImageSet = false;
	}
</script>

<div class="flex flex-col items-start md:flex-row md:items-center" {...restProps}>
	<Label class="w-52" for={id}>{label}</Label>
	<FileInput {id} variant="secondary" {accept} onchange={onImageChange}>
		<div
			class={cn('group/image relative flex items-center rounded transition-colors', {
				'bg-[#F5F5F5]': forceColorScheme === 'light',
				'bg-[#262626]': forceColorScheme === 'dark',
				'bg-muted': !forceColorScheme
			})}
		>
			{#if !isImageSet}
				<div
					class={cn(
						'flex h-full w-full items-center justify-center p-3 transition-opacity duration-200',
						'group-hover/image:opacity-10 group-has-[button:hover]/image:opacity-100',
						imageClass
					)}
				>
					<LucideImageOff class="text-muted-foreground" />
				</div>
			{:else}
				<img
					class={cn(
						'h-full w-full rounded object-cover p-3 transition-opacity duration-200',
						'group-hover/image:opacity-10 group-has-[button:hover]/image:opacity-100',
						imageClass
					)}
					src={imageDataURL}
					alt={label}
					onerror={() => (isImageSet = false)}
				/>
			{/if}
			<LucideUpload
				class={cn(
					'absolute top-1/2 left-1/2 size-5 -translate-x-1/2 -translate-y-1/2 transform font-medium opacity-0 transition-opacity duration-200',
					'group-hover/image:opacity-100 group-has-[button:hover]/image:opacity-0',
					{
						'text-black': forceColorScheme === 'light',
						'text-white': forceColorScheme === 'dark'
					}
				)}
			/>
			{#if isResetable && isImageSet}
				<Button
					size="icon"
					onclick={(e) => {
						e.stopPropagation();
						onReset();
					}}
					class="absolute -top-2 -right-2 size-6 rounded-full shadow-md"
				>
					<LucideX class="size-3" />
				</Button>
			{/if}
		</div>
	</FileInput>
</div>
