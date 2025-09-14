<script lang="ts">
	import FileInput from '$lib/components/form/file-input.svelte';
	import FormInput from '$lib/components/form/form-input.svelte';
	import SwitchWithLabel from '$lib/components/form/switch-with-label.svelte';
	import ImageBox from '$lib/components/image-box.svelte';
	import { Button } from '$lib/components/ui/button';
	import Label from '$lib/components/ui/label/label.svelte';
	import { m } from '$lib/paraglide/messages';
	import type {
		OidcClient,
		OidcClientCreateWithLogo,
		OidcClientUpdateWithLogo
	} from '$lib/types/oidc.type';
	import { cachedOidcClientLogo } from '$lib/utils/cached-image-util';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { cn } from '$lib/utils/style';
	import { callbackUrlSchema, emptyToUndefined, optionalUrl } from '$lib/utils/zod-util';
	import { LucideChevronDown, LucideX } from '@lucide/svelte';
	import { slide } from 'svelte/transition';
	import { z } from 'zod/v4';
	import FederatedIdentitiesInput from './federated-identities-input.svelte';
	import OidcCallbackUrlInput from './oidc-callback-url-input.svelte';
	import * as DropdownButton from '$lib/components/ui/dropdown-button';

	let {
		callback,
		existingClient,
		mode
	}: {
		existingClient?: OidcClient;
		callback: (client: OidcClientCreateWithLogo | OidcClientUpdateWithLogo) => Promise<boolean>;
		mode: 'create' | 'update';
	} = $props();
	let isLoading = $state(false);
	let showAdvancedOptions = $state(false);
	let logo = $state<File | null | undefined>();
	let logoDataURL: string | null = $state(
		existingClient?.hasLogo ? cachedOidcClientLogo.getUrl(existingClient!.id) : null
	);
	let isUsingSelfhostedIcon = $state(false);
	let isUsingUrl = $state(false);
	let clientLogoCleared = $state(false);

	const client = {
		id: '',
		name: existingClient?.name || '',
		callbackURLs: existingClient?.callbackURLs || [],
		logoutCallbackURLs: existingClient?.logoutCallbackURLs || [],
		isPublic: existingClient?.isPublic || false,
		pkceEnabled: existingClient?.pkceEnabled || false,
		requiresReauthentication: existingClient?.requiresReauthentication || false,
		launchURL: existingClient?.launchURL || '',
		logoUrl: existingClient?.logoUrl || '',
		credentials: {
			federatedIdentities: existingClient?.credentials?.federatedIdentities || []
		}
	};

	const formSchema = z.object({
		id: emptyToUndefined(
			z
				.string()
				.min(2)
				.max(128)
				.regex(/^[a-zA-Z0-9_-]+$/, {
					message: m.invalid_client_id()
				})
				.optional()
		),
		name: z.string().min(2).max(50),
		callbackURLs: z.array(callbackUrlSchema).default([]),
		logoutCallbackURLs: z.array(callbackUrlSchema).default([]),
		isPublic: z.boolean(),
		pkceEnabled: z.boolean(),
		requiresReauthentication: z.boolean(),
		launchURL: optionalUrl,
		logoUrl: optionalUrl,
		credentials: z.object({
			federatedIdentities: z.array(
				z.object({
					issuer: z.url(),
					subject: z.string().optional(),
					audience: z.string().optional(),
					jwks: z.url().optional().or(z.literal(''))
				})
			)
		})
	});

	type FormSchema = typeof formSchema;
	const { inputs, errors, ...form } = createForm<FormSchema>(formSchema, client);

	async function onSubmit() {
		const data = form.validate();
		if (!data) return;
		isLoading = true;

		const logoUrl = isUsingUrl ? ($inputs.logoUrl?.value as string | undefined)?.trim() : undefined;

		const success = await callback({
			...data,
			logo: isUsingUrl ? null : logo,
			logoUrl
		});

		if (success && isUsingUrl && existingClient) {
			logoDataURL = cachedOidcClientLogo.getUrl(existingClient.id);
		}

		if (success && !existingClient) form.reset();
		isLoading = false;
	}

	function onLogoChange(e: Event) {
		const file = (e.target as HTMLInputElement).files?.[0] || null;
		if (file) {
			logo = file;
			isUsingUrl = false;
			isUsingSelfhostedIcon = false;
			$inputs.logoUrl && ($inputs.logoUrl.value = '');
			clientLogoCleared = false;
			const reader = new FileReader();
			reader.onload = (event) => (logoDataURL = event.target?.result as string);
			reader.readAsDataURL(file);
		}
	}

	function resetLogo() {
		logo = null;
		logoDataURL = null;
		isUsingUrl = false;
		$inputs.logoUrl && ($inputs.logoUrl.value = '');
		clientLogoCleared = true;
		isUsingSelfhostedIcon = false;
	}

	function getFederatedIdentityErrors(errors: z.ZodError<any> | undefined) {
		return errors?.issues
			.filter((e) => e.path[0] == 'credentials' && e.path[1] == 'federatedIdentities')
			.map((e) => {
				e.path.splice(0, 2);
				return e;
			});
	}
</script>

<form onsubmit={preventDefault(onSubmit)}>
	<div class="grid grid-cols-1 gap-x-3 gap-y-7 sm:flex-row md:grid-cols-2">
		<FormInput
			label={m.name()}
			class="w-full"
			description={m.client_name_description()}
			bind:input={$inputs.name}
		/>
		<FormInput
			label={m.client_launch_url()}
			description={m.client_launch_url_description()}
			class="w-full"
			bind:input={$inputs.launchURL}
		/>
		<OidcCallbackUrlInput
			label={m.callback_urls()}
			description={m.callback_url_description()}
			class="w-full"
			bind:callbackURLs={$inputs.callbackURLs.value}
			bind:error={$inputs.callbackURLs.error}
		/>
		<OidcCallbackUrlInput
			label={m.logout_callback_urls()}
			description={m.logout_callback_url_description()}
			class="w-full"
			bind:callbackURLs={$inputs.logoutCallbackURLs.value}
			bind:error={$inputs.logoutCallbackURLs.error}
		/>
		<SwitchWithLabel
			id="public-client"
			label={m.public_client()}
			description={m.public_clients_description()}
			bind:checked={$inputs.isPublic.value}
		/>
		<SwitchWithLabel
			id="pkce"
			label={m.pkce()}
			description={m.public_key_code_exchange_is_a_security_feature_to_prevent_csrf_and_authorization_code_interception_attacks()}
			bind:checked={$inputs.pkceEnabled.value}
		/>
		<SwitchWithLabel
			id="requires-reauthentication"
			label={m.requires_reauthentication()}
			description={m.requires_users_to_authenticate_again_on_each_authorization()}
			bind:checked={$inputs.requiresReauthentication.value}
		/>
	</div>
	<div class="mt-8">
		<Label for="logo">{m.logo()}</Label>

		<div class="mt-2 space-y-4">
			{#if logoDataURL}
				<div class="flex items-start gap-4">
					<div class="relative shrink-0">
						<ImageBox
							class="size-24"
							src={logoDataURL}
							alt={m.name_logo({ name: $inputs.name.value })}
						/>

						<Button
							variant="destructive"
							size="icon"
							onclick={resetLogo}
							class="absolute -right-2 -top-2 size-6 rounded-full shadow-md"
						>
							<LucideX class="size-3" />
						</Button>
					</div>

					<div class="flex w-full max-w-xl flex-col gap-3">
						{#if isUsingUrl}
							<FormInput
								label="Logo URL"
								placeholder="https://example.com/icon.svg"
								class="w-full"
								bind:input={$inputs.logoUrl}
							/>
							<p class="text-muted-foreground text-[0.8rem]">
								Paste a direct image URL (svg, png, webp). Find icons at
								<a
									class="underline"
									rel="noreferrer"
									target="_blank"
									href="https://github.com/selfhst/icons">Selfh.st Icons</a
								>
								or
								<a
									class="underline"
									rel="noreferrer"
									target="_blank"
									href="https://github.com/homarr-labs/dashboard-icons">Dashboard Icons</a
								>.
							</p>

							<div class="flex gap-2">
								<Button
									variant="secondary"
									onclick={() => {
										isUsingUrl = false;
										$inputs.logoUrl && ($inputs.logoUrl.value = '');
									}}
								>
									Switch to upload
								</Button>
							</div>
						{:else}
							<!-- Upload mode + DropdownButton -->
							<div class="flex items-center gap-2">
								<DropdownButton.DropdownRoot>
									<DropdownButton.Root>
										<FileInput
											id="logo"
											variant="secondary"
											accept="image/png, image/jpeg, image/svg+xml"
											onchange={onLogoChange}
										>
											<DropdownButton.Main class="min-w-32">
												{m.change_logo()}
											</DropdownButton.Main>
										</FileInput>

										<DropdownButton.DropdownTrigger>
											<DropdownButton.Trigger class="border-l" />
										</DropdownButton.DropdownTrigger>
									</DropdownButton.Root>

									<DropdownButton.Content align="end">
										<DropdownButton.Item onclick={() => (isUsingUrl = true)}>
											Use URL
										</DropdownButton.Item>
									</DropdownButton.Content>
								</DropdownButton.DropdownRoot>
							</div>
						{/if}
					</div>
				</div>
			{:else}
				<div class="flex flex-col gap-3">
					{#if isUsingUrl}
						<FormInput
							label="Logo URL"
							placeholder="https://example.com/icon.svg"
							class="w-full"
							bind:input={$inputs.logoUrl}
						/>
						<p class="text-muted-foreground text-[0.8rem]">
							Paste a direct image URL (svg, png, webp). Find icons at
							<a
								class="underline"
								rel="noreferrer"
								target="_blank"
								href="https://github.com/selfhst/icons">Selfh.st Icons</a
							>
							or
							<a
								class="underline"
								rel="noreferrer"
								target="_blank"
								href="https://github.com/homarr-labs/dashboard-icons">Dashboard Icons</a
							>.
						</p>

						<div class="flex gap-2">
							<Button
								variant="secondary"
								onclick={() => {
									isUsingUrl = false;
									$inputs.logoUrl && ($inputs.logoUrl.value = '');
								}}
							>
								Switch to upload
							</Button>
						</div>
					{:else}
						<div class="flex flex-wrap items-center gap-2">
							<DropdownButton.DropdownRoot>
								<DropdownButton.Root>
									<FileInput
										id="logo"
										variant="secondary"
										accept="image/png, image/jpeg, image/svg+xml"
										onchange={onLogoChange}
									>
										<DropdownButton.Main class="min-w-32">
											{m.upload_logo()}
										</DropdownButton.Main>
									</FileInput>

									<DropdownButton.DropdownTrigger>
										<DropdownButton.Trigger class="border-l" />
									</DropdownButton.DropdownTrigger>
								</DropdownButton.Root>

								<DropdownButton.Content align="end">
									<DropdownButton.Item onclick={() => (isUsingUrl = true)}>
										Use URL
									</DropdownButton.Item>
								</DropdownButton.Content>
							</DropdownButton.DropdownRoot>
						</div>
					{/if}
				</div>
			{/if}
		</div>
	</div>

	{#if showAdvancedOptions}
		<div class="mt-7 flex flex-col gap-y-7 md:col-span-2" transition:slide={{ duration: 200 }}>
			{#if mode == 'create'}
				<FormInput
					label={m.client_id()}
					placeholder={m.generated()}
					class="w-full md:w-1/2"
					description={m.custom_client_id_description()}
					bind:input={$inputs.id}
				/>
			{/if}
			<FederatedIdentitiesInput
				client={existingClient}
				bind:federatedIdentities={$inputs.credentials.value.federatedIdentities}
				errors={getFederatedIdentityErrors($errors)}
			/>
		</div>
	{/if}

	<div class="relative mt-5 flex justify-center">
		<Button
			variant="ghost"
			class="text-muted-foreground"
			onclick={() => (showAdvancedOptions = !showAdvancedOptions)}
		>
			{showAdvancedOptions ? m.hide_advanced_options() : m.show_advanced_options()}
			<LucideChevronDown
				class={cn(
					'size-5 transition-transform duration-200',
					showAdvancedOptions && 'rotate-180 transform'
				)}
			/>
		</Button>
		<Button {isLoading} type="submit" class="absolute right-0">{m.save()}</Button>
	</div>
</form>
