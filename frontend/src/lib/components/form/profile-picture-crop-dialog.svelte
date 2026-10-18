<script lang="ts">
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Dialog from '#lib/components/ui/dialog/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { LucideZoomIn, LucideZoomOut } from '@lucide/svelte';

	let {
		file = $bindable(),
		onSave
	}: {
		file: File | null;
		onSave: (image: File) => Promise<void>;
	} = $props();

	const OUTPUT_SIZE = 512;
	const MAX_ZOOM = 4;
	const CIRCLE_INSET = 24;

	let img = $state<HTMLImageElement>();
	let natural = $state({ w: 0, h: 0 });
	let stage = $state({ w: 0, h: 0 });
	let zoom = $state(1);
	let offset = $state({ x: 0, y: 0 });
	let drag: { x: number; y: number; ox: number; oy: number } | null = null;
	let saving = $state(false);

	const src = $derived(file ? URL.createObjectURL(file) : '');
	$effect(() => {
		const url = src;
		return () => URL.revokeObjectURL(url);
	});

	const diameter = $derived(Math.max(0, Math.min(stage.w, stage.h) - CIRCLE_INSET * 2));
	const scale = $derived(natural.w ? (diameter / Math.min(natural.w, natural.h)) * zoom : 0);

	// The offset is clamped on read so the photo always covers the circle, whatever the zoom or stage size
	const maxX = $derived(Math.max(0, (natural.w * scale - diameter) / 2));
	const maxY = $derived(Math.max(0, (natural.h * scale - diameter) / 2));
	const x = $derived(Math.max(-maxX, Math.min(maxX, offset.x)));
	const y = $derived(Math.max(-maxY, Math.min(maxY, offset.y)));

	function reset() {
		natural = { w: 0, h: 0 };
		zoom = 1;
		offset = { x: 0, y: 0 };
	}

	function setZoom(value: number) {
		zoom = Math.max(1, Math.min(MAX_ZOOM, value));
	}

	function onKeydown(e: KeyboardEvent) {
		const step = e.shiftKey ? 40 : 8;
		const moves: Record<string, [number, number]> = {
			ArrowLeft: [-step, 0],
			ArrowRight: [step, 0],
			ArrowUp: [0, -step],
			ArrowDown: [0, step]
		};
		const move = moves[e.key];
		if (!move) return;
		e.preventDefault();
		offset = { x: x + move[0], y: y + move[1] };
	}

	async function save() {
		if (!img || !scale) return;

		// Map the circle's bounding square from stage pixels back to source pixels
		const sx = (natural.w * scale - diameter) / 2 - x;
		const sy = (natural.h * scale - diameter) / 2 - y;
		const size = diameter / scale;

		const canvas = document.createElement('canvas');
		canvas.width = canvas.height = OUTPUT_SIZE;
		canvas
			.getContext('2d')!
			.drawImage(img, sx / scale, sy / scale, size, size, 0, 0, OUTPUT_SIZE, OUTPUT_SIZE);
		const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/png'));
		if (!blob) return;

		saving = true;
		try {
			await onSave(new File([blob], 'profile-picture.png', { type: 'image/png' }));
			file = null;
		} finally {
			saving = false;
		}
	}

	// Chromium and Firefox cannot decode HEIC, so those photos skip the crop step and upload as they are
	async function onDecodeError() {
		const original = file;
		file = null;
		if (original) await onSave(original);
	}
</script>

<Dialog.Root
	open={!!file}
	onOpenChange={(open) => {
		if (!open && !saving) file = null;
	}}
	onOpenChangeComplete={(open) => !open && reset()}
>
	<Dialog.Content class="max-w-(--dialog-md)">
		<Dialog.Header>
			<Dialog.Title>{m.adjust_your_photo()}</Dialog.Title>
			<Dialog.Description>{m.adjust_your_photo_description()}</Dialog.Description>
		</Dialog.Header>

		<!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions -->
		<div
			role="application"
			aria-label={m.photo_crop_area()}
			tabindex="0"
			class="bg-well-dark focus-visible:outline-ring relative h-72 cursor-grab touch-none overflow-hidden rounded-lg select-none focus-visible:outline-2 focus-visible:outline-offset-2 active:cursor-grabbing"
			bind:clientWidth={stage.w}
			bind:clientHeight={stage.h}
			onpointerdown={(e) => {
				e.currentTarget.setPointerCapture(e.pointerId);
				drag = { x: e.clientX, y: e.clientY, ox: x, oy: y };
			}}
			onpointermove={(e) => {
				if (drag) offset = { x: drag.ox + e.clientX - drag.x, y: drag.oy + e.clientY - drag.y };
			}}
			onpointerup={() => (drag = null)}
			onpointercancel={() => (drag = null)}
			onwheel={(e) => {
				e.preventDefault();
				setZoom(zoom - e.deltaY * 0.002);
			}}
			onkeydown={onKeydown}
		>
			{#if src}
				<img
					bind:this={img}
					{src}
					alt=""
					draggable="false"
					class="absolute top-1/2 left-1/2 max-w-none transition-opacity duration-(--dur-base)"
					class:opacity-0={!scale}
					style:width="{natural.w * scale}px"
					style:transform="translate(calc(-50% + {x}px), calc(-50% + {y}px))"
					onload={() => (natural = { w: img!.naturalWidth, h: img!.naturalHeight })}
					onerror={onDecodeError}
				/>
			{/if}
			<!-- The spread shadow dims the photo outside the circle while keeping it visible for context -->
			<div
				class="pointer-events-none absolute top-1/2 left-1/2 -translate-1/2 rounded-full shadow-[0_0_0_9999px_rgb(0_0_0/0.6)] ring-2 ring-white/80"
				style:width="{diameter}px"
				style:height="{diameter}px"
			></div>
		</div>

		<div class="flex items-center gap-2">
			<Button
				variant="ghost"
				size="icon-sm"
				aria-label={m.zoom_out()}
				disabled={zoom <= 1}
				onclick={() => setZoom(zoom - 0.25)}
			>
				<LucideZoomOut />
			</Button>
			<input
				type="range"
				min="1"
				max={MAX_ZOOM}
				step="0.01"
				aria-label={m.zoom()}
				class="accent-fg h-1 flex-1 cursor-pointer"
				bind:value={zoom}
			/>
			<Button
				variant="ghost"
				size="icon-sm"
				aria-label={m.zoom_in()}
				disabled={zoom >= MAX_ZOOM}
				onclick={() => setZoom(zoom + 0.25)}
			>
				<LucideZoomIn />
			</Button>
		</div>

		<Dialog.Footer>
			<Button variant="secondary" disabled={saving} onclick={() => (file = null)}>
				{m.cancel()}
			</Button>
			<Button isLoading={saving} disabled={!scale} onclick={save}>{m.save()}</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
