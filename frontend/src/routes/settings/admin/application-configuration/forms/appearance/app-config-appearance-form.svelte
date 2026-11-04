<script lang="ts">
	import AppearancePreview from '#lib/components/appearance/appearance-preview.svelte';
	import {
		appearance,
		trackAppearance,
		defaultAppearance,
		textSizes,
		fonts,
		layouts,
		radii,
		type Font,
		type Layout
	} from '#lib/components/appearance/appearance.svelte.ts';
	import { m } from '#lib/paraglide/messages.js';
	import type { AllAppConfig } from '#lib/types/application-configuration.type.ts';
	import { deepEqual } from '#lib/utils/form-util.ts';
	import { Badge } from '@pocket-id/ui/badge';
	import { Button } from '@pocket-id/ui/button';
	import { Input } from '@pocket-id/ui/input';
	import { Switch } from '@pocket-id/ui/switch';
	import { cn } from '@pocket-id/ui/utils';
	import { RadioGroup } from 'bits-ui';
	import CheckIcon from '~icons/solar/check-circle-linear';
	import RestartIcon from '~icons/solar/restart-bold';
	import UploadIcon from '~icons/solar/upload-minimalistic-bold';
	import ImageSections from './image-sections.svelte';
	import PreviewFigure from './preview-figure.svelte';
	import PreviewSection, { tileClass } from './preview-section.svelte';
	import SurfacesSection from './surfaces-section.svelte';

	let {
		callback
	}: {
		callback: (appConfig: Partial<AllAppConfig>) => Promise<void>;
	} = $props();

	trackAppearance(callback);

	const uid = $props.id();

	const customFont = $derived(appearance.customFont);
	const isDefault = $derived(deepEqual(appearance.value, defaultAppearance));
	const shownFonts = $derived(customFont ? fonts : fonts.filter((f) => f.value !== 'custom'));

	// The draft only declares the custom font while it uses it, so the custom tile keeps the font loaded on its own
	$effect(() => {
		if (!customFont) return;
		const face = new FontFace('Uploaded Font', `url("${customFont.url}")`);
		document.fonts.add(face);
		return () => document.fonts.delete(face);
	});

	function uploadFont(e: Event & { currentTarget: HTMLInputElement }) {
		const file = e.currentTarget.files?.[0];
		e.currentTarget.value = '';
		if (file) appearance.uploadFont(file);
	}

	const fontFamily = (value: Font['kind']) =>
		value === 'custom'
			? "'Uploaded Font', 'Young Serif', serif"
			: fonts.find((f) => f.value === value)!.family;
</script>

<div class="flex w-full flex-col">
	<!-- Layout and Colors share one preview, which stays beside both while they scroll past -->
	<!-- On phones it sits between them, on desktop it spans both rows of the second column -->
	<!-- The tab row above already divides, so the first section starts without a hairline or top padding -->
	<div class="grid gap-x-14 lg:grid-cols-[minmax(0,9fr)_minmax(0,11fr)]">
		<PreviewSection title="Layout" class="pt-0 shadow-none lg:col-start-1 lg:row-start-1">
			<RadioGroup.Root
				aria-label="Layout"
				class="flex flex-col gap-2"
				bind:value={() => appearance.value.layout, (v) => (appearance.value.layout = v as Layout)}
			>
				{#each layouts as layout (layout.value)}
					<RadioGroup.Item
						value={layout.value}
						class={cn(tileClass, 'flex items-center gap-4 p-2.5 pe-4')}
					>
						{#snippet children({ checked })}
							<AppearancePreview layout={layout.value} class="w-24 shrink-0" />
							<span class="text-control min-w-0 flex-1 font-medium">{layout.label}</span>
							<CheckIcon
								class={cn(
									'text-primary-control size-5 shrink-0 transition-opacity',
									!checked && 'opacity-0'
								)}
							/>
						{/snippet}
					</RadioGroup.Item>
				{/each}
			</RadioGroup.Root>
		</PreviewSection>

		<!-- The reset sits on the row of the Layout heading, so the preview below it starts level with the layout cards -->
		<div class="flex flex-col gap-7 lg:col-start-2 lg:row-span-2 lg:row-start-1 lg:mb-12">
			<div class="flex justify-end">
				<Button
					variant="ghost"
					size="sm"
					disabled={isDefault}
					onclick={() => (appearance.value = structuredClone(defaultAppearance))}
				>
					<RestartIcon />
					{m.reset_to_default()}
				</Button>
			</div>
			<PreviewFigure class="self-stretch" stageClass="min-h-0 p-0 sm:p-0">
				<AppearancePreview flush />
			</PreviewFigure>
		</div>

		<SurfacesSection class="max-lg:shadow-none lg:col-start-1 lg:row-start-2" />
	</div>

	<PreviewSection title="Typography" preview={typePreview}>
		<div class="flex flex-col gap-3">
			<p id="{uid}-font" class="text-control text-fg font-medium">Title font</p>
			<!-- The upload card always takes the last column, so every card is as wide as the ones below -->
			<div class="grid grid-cols-3 gap-2">
				<RadioGroup.Root
					aria-labelledby="{uid}-font"
					orientation="horizontal"
					class={cn('grid gap-2', customFont ? 'col-span-2 grid-cols-2' : 'grid-cols-1')}
					bind:value={
						() => appearance.value.font.kind, (kind) => appearance.setFont(kind as Font['kind'])
					}
				>
					{#each shownFonts as option (option.value)}
						<RadioGroup.Item
							value={option.value}
							class={cn(tileClass, 'flex h-28 flex-col justify-between p-3.5')}
						>
							<span class="text-[34px] leading-none" style:font-family={fontFamily(option.value)}
								>Aa</span
							>
							{@const name = option.value === 'custom' ? customFont?.name : option.label}
							<span class="text-caption text-fg-muted w-full truncate" title={name}>{name}</span>
						</RadioGroup.Item>
					{/each}
				</RadioGroup.Root>
				<label
					class="border-border text-fg-muted hover:text-fg hover:bg-fill-row-hover has-focus-visible:outline-ring flex h-28 cursor-pointer flex-col justify-between rounded-md border border-dashed p-3.5 transition-colors duration-(--dur-fast) has-focus-visible:outline-2 has-focus-visible:outline-offset-2"
				>
					<input
						type="file"
						class="sr-only"
						accept=".woff2,.woff,.ttf,.otf"
						onchange={uploadFont}
					/>
					<UploadIcon class="size-5" />
					<span class="text-caption text-fg font-medium"
						>{customFont ? 'Upload another' : 'Upload a font'}</span
					>
				</label>
			</div>
		</div>

		<div class="flex flex-col gap-3">
			<p id="{uid}-size" class="text-control text-fg font-medium">Text size</p>
			<RadioGroup.Root
				aria-labelledby="{uid}-size"
				orientation="horizontal"
				class="grid grid-cols-3 gap-2"
				bind:value={
					() => String(appearance.value.textSize), (v) => (appearance.value.textSize = Number(v))
				}
			>
				{#each textSizes as size (size.value)}
					<RadioGroup.Item
						value={String(size.value)}
						class={cn(tileClass, 'flex flex-col items-start gap-3 p-3.5')}
					>
						<span
							aria-hidden="true"
							class="flex h-6 items-end leading-none"
							style:font-size="{size.value + 6}px">Aa</span
						>
						<span class="text-control font-medium">{size.label}</span>
					</RadioGroup.Item>
				{/each}
			</RadioGroup.Root>
		</div>
	</PreviewSection>

	<PreviewSection title="Corner radius" preview={radiusPreview}>
		<RadioGroup.Root
			aria-label="Corner radius"
			orientation="horizontal"
			class="grid grid-cols-5 gap-2"
			bind:value={
				() => String(appearance.value.radius), (v) => (appearance.value.radius = Number(v))
			}
		>
			{#each radii as option (option.value)}
				<RadioGroup.Item
					value={String(option.value)}
					class={cn(tileClass, 'flex flex-col items-center gap-3 px-1 pt-4 pb-3')}
				>
					<span
						aria-hidden="true"
						class="border-fg size-8 border-t-2 border-l-2"
						style:border-top-left-radius="{option.value * 16}px"
					></span>
					<span class="text-caption text-fg-muted w-full truncate text-center">{option.label}</span>
				</RadioGroup.Item>
			{/each}
		</RadioGroup.Root>
	</PreviewSection>

	<ImageSections />
</div>

{#snippet typePreview()}
	<div
		class="bg-surface border-border w-full max-w-md rounded-xl border p-6 shadow-[0_1px_2px_rgb(0_0_0/0.04)]"
	>
		<h3 class="font-display text-display-m text-fg">Sign-in methods</h3>
		<p class="text-body text-fg-muted mt-2">
			Passkeys let people sign in with a fingerprint, their face or a screen lock, so there is
			nothing to remember and nothing to phish.
		</p>
		<div class="mt-4 flex flex-col">
			{#each [{ title: 'Passkeys', text: 'Two registered, last used today on a MacBook.' }, { title: 'Login codes', text: 'One-time codes for people who lost their device.' }] as row (row.title)}
				<div
					class="flex flex-col gap-0.5 py-3 shadow-[inset_0_-1px_0_var(--border)] last:shadow-none"
				>
					<p class="text-heading text-fg">{row.title}</p>
					<p class="text-caption text-fg-muted">{row.text}</p>
				</div>
			{/each}
		</div>
	</div>
{/snippet}

{#snippet radiusPreview()}
	<div
		class="bg-surface border-border flex w-full max-w-sm flex-col gap-4 rounded-xl border p-5 shadow-[0_1px_2px_rgb(0_0_0/0.04)]"
	>
		<div class="flex items-center justify-between gap-3">
			<p class="text-heading text-fg">Invite a user</p>
			<Badge>Admin</Badge>
		</div>
		<Input aria-label="Email" value="alex@example.com" />
		<div class="text-control text-fg flex items-center gap-3">
			<Switch checked aria-label="Send an invite email" />
			Send an invite email
		</div>
		<div class="flex justify-end gap-2">
			<Button variant="ghost">Cancel</Button>
			<Button>Send invite</Button>
		</div>
	</div>
{/snippet}
