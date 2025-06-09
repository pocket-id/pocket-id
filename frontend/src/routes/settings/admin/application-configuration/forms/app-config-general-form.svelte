<script lang="ts">
	import SwitchWithLabel from '$lib/components/form/switch-with-label.svelte';
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Label } from '$lib/components/ui/label/index.js';
	import * as RadioGroup from '$lib/components/ui/radio-group/index.js';
	import { m } from '$lib/paraglide/messages';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { AllAppConfig } from '$lib/types/application-configuration';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { applyAccentColor } from '$lib/utils/accent-color-util';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod/v4';
	import { Check, Plus } from '@lucide/svelte';
	import CustomColorDialog from './custom-color-dialog.svelte';

	let {
		callback,
		appConfig
	}: {
		appConfig: AllAppConfig;
		callback: (appConfig: Partial<AllAppConfig>) => Promise<void>;
	} = $props();

	let isLoading = $state(false);
	let showCustomColorDialog = $state(false);

	const updatedAppConfig = {
		appName: appConfig.appName,
		sessionDuration: appConfig.sessionDuration,
		emailsVerified: appConfig.emailsVerified,
		allowOwnAccountEdit: appConfig.allowOwnAccountEdit,
		disableAnimations: appConfig.disableAnimations,
		accentColor: appConfig.accentColor || 'default'
	};

	const formSchema = z.object({
		appName: z.string().min(2).max(30),
		sessionDuration: z.number().min(1).max(43200),
		emailsVerified: z.boolean(),
		allowOwnAccountEdit: z.boolean(),
		disableAnimations: z.boolean(),
		accentColor: z.string()
	});

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

	const { inputs, ...form } = createForm<typeof formSchema>(formSchema, updatedAppConfig);

	// Check if current accent color is a custom color (not in predefined list)
	let isCustomColor = $derived(
		!accentColors.some((color) => color.value === $inputs.accentColor.value)
	);

	async function onSubmit() {
		const data = form.validate();
		if (!data) return;
		isLoading = true;

		// Apply accent color immediately
		applyAccentColor(data.accentColor);

		await callback(data).finally(() => (isLoading = false));
		toast.success(m.application_configuration_updated_successfully());
	}

	function handleAccentColorChange(accentValue: string) {
		$inputs.accentColor.value = accentValue;
		applyAccentColor(accentValue);
	}

	function openCustomColorDialog() {
		showCustomColorDialog = true;
	}

	function handleCustomColorApply(color: string) {
		handleAccentColorChange(color);
	}
</script>

<form onsubmit={preventDefault(onSubmit)}>
	<fieldset class="flex flex-col gap-5" disabled={$appConfigStore.uiConfigDisabled}>
		<div class="flex flex-col gap-5">
			<FormInput label={m.application_name()} bind:input={$inputs.appName} />
			<FormInput
				label={m.session_duration()}
				type="number"
				description={m.the_duration_of_a_session_in_minutes_before_the_user_has_to_sign_in_again()}
				bind:input={$inputs.sessionDuration}
			/>

			<SwitchWithLabel
				id="self-account-editing"
				label={m.enable_self_account_editing()}
				description={m.whether_the_users_should_be_able_to_edit_their_own_account_details()}
				bind:checked={$inputs.allowOwnAccountEdit.value}
			/>
			<SwitchWithLabel
				id="emails-verified"
				label={m.emails_verified()}
				description={m.whether_the_users_email_should_be_marked_as_verified_for_the_oidc_clients()}
				bind:checked={$inputs.emailsVerified.value}
			/>
			<SwitchWithLabel
				id="disable-animations"
				label={m.disable_animations()}
				description={m.turn_off_ui_animations()}
				bind:checked={$inputs.disableAnimations.value}
			/>

			<!-- Accent Color Picker -->
			<div class="space-y-3">
				<div class="space-y-1">
					<Label
						class="text-sm leading-none font-medium peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
					>
						{m.accent_color()}
					</Label>
					<p class="text-muted-foreground text-sm">
						{m.select_an_accent_color_to_customize_the_appearance_of_pocket_id()}
					</p>
				</div>

				<RadioGroup.Root
					class="flex flex-wrap gap-3"
					value={isCustomColor ? 'custom' : $inputs.accentColor.value}
					onValueChange={(value) => {
						if (value === 'custom') {
							openCustomColorDialog();
						} else {
							handleAccentColorChange(value);
						}
					}}
				>
					{#each accentColors as accent}
						<div class="relative">
							<RadioGroup.Item value={accent.value} id={accent.value} class="sr-only" />
							<Label for={accent.value} class="group cursor-pointer">
								<div
									class="relative h-10 w-10 rounded-full border-2 transition-all duration-200 {$inputs
										.accentColor.value === accent.value
										? 'border-primary ring-primary ring-2 ring-offset-2'
										: 'group-hover:border-primary group-hover:ring-primary border-gray-200 group-hover:ring-1 group-hover:ring-offset-1'}"
									style="background-color: {accent.color}"
									title={accent.label}
								>
									{#if $inputs.accentColor.value === accent.value}
										<div class="absolute inset-0 flex items-center justify-center">
											<Check class="size-4 text-white drop-shadow-sm" />
										</div>
									{/if}
								</div>
								<div
									class="text-muted-foreground group-hover:text-foreground mt-1 text-center text-xs transition-colors"
								>
									{accent.label}
								</div>
							</Label>
						</div>
					{/each}

					<!-- Custom Color Option -->
					<div class="relative">
						<RadioGroup.Item value="custom" id="custom" class="sr-only" />
						<Label for="custom" class="group cursor-pointer">
							<div
								class="relative h-10 w-10 rounded-full border-2 transition-all duration-200 {isCustomColor
									? 'border-primary ring-primary ring-2 ring-offset-2'
									: 'group-hover:border-primary group-hover:ring-primary border-gray-200 group-hover:ring-1 group-hover:ring-offset-1'}"
								style="background-color: {isCustomColor
									? $inputs.accentColor.value
									: 'transparent'}"
								title="Custom Color"
							>
								{#if isCustomColor}
									<div class="absolute inset-0 flex items-center justify-center">
										<Check class="size-4 text-white drop-shadow-sm" />
									</div>
								{:else}
									<div
										class="bg-muted absolute inset-0 flex items-center justify-center rounded-full border-2 border-dashed border-gray-300"
									>
										<Plus class="text-muted-foreground size-4" />
									</div>
								{/if}
							</div>
							<div
								class="text-muted-foreground group-hover:text-foreground mt-1 text-center text-xs transition-colors"
							>
								Custom
							</div>
						</Label>
					</div>
				</RadioGroup.Root>
			</div>
		</div>
		<div class="mt-5 flex justify-end">
			<Button {isLoading} type="submit">{m.save()}</Button>
		</div>
	</fieldset>
</form>

<CustomColorDialog bind:open={showCustomColorDialog} onApply={handleCustomColorApply} />
