<script lang="ts">
	import FileInput from '$lib/components/form/file-input.svelte';
	import FormInput from '$lib/components/form/form-input.svelte';
	import SwitchWithLabel from '$lib/components/form/switch-with-label.svelte';
	import ImageBox from '$lib/components/image-box.svelte';
	import { Button } from '$lib/components/ui/button';
	import Label from '$lib/components/ui/label/label.svelte';
	import { m } from '$lib/paraglide/messages';
	import type { OidcClient, OidcClientCreateWithLogo } from '$lib/types/oidc.type';
	import { cachedOidcClientLogo } from '$lib/utils/cached-image-util';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { cn } from '$lib/utils/style';
	import { optionalUrl } from '$lib/utils/zod-util';
	import { LucideChevronDown } from '@lucide/svelte';
	import { slide } from 'svelte/transition';
	import { z } from 'zod/v4';
	import FederatedIdentitiesInput from './federated-identities-input.svelte';
	import OidcCallbackUrlInput from './oidc-callback-url-input.svelte';

	let {
		callback,
		existingClient
	}: {
		existingClient?: OidcClient;
		callback: (user: OidcClientCreateWithLogo) => Promise<boolean>;
	} = $props();

	let isLoading = $state(false);
	let showAdvancedOptions = $state(false);
	let logo = $state<File | null | undefined>();
	let logoDataURL: string | null = $state(
		existingClient?.hasLogo ? cachedOidcClientLogo.getUrl(existingClient!.id) : null
	);

	const client = {
		name: existingClient?.name || '',
		callbackURLs: existingClient?.callbackURLs || [],
		logoutCallbackURLs: existingClient?.logoutCallbackURLs || [],
		isPublic: existingClient?.isPublic || false,
		pkceEnabled: existingClient?.pkceEnabled || false,
		requiresReauthentication: existingClient?.requiresReauthentication || false,
		launchURL: existingClient?.launchURL || '',
		credentials: {
			federatedIdentities: existingClient?.credentials?.federatedIdentities || []
		}
	};

	const formSchema = z.object({
		name: z.string().min(2).max(50),
		callbackURLs: z.array(z.string().nonempty()).default([]),
		logoutCallbackURLs: z.array(z.string().nonempty()),
		isPublic: z.boolean(),
		pkceEnabled: z.boolean(),
		requiresReauthentication: z.boolean(),
		launchURL: optionalUrl,
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
		const success = await callback({
			...data,
			logo
		});
		// Reset form if client was successfully created
		if (success && !existingClient) form.reset();
		isLoading = false;
	}

	function onLogoChange(e: Event) {
		const file = (e.target as HTMLInputElement).files?.[0] || null;
		if (file) {
			logo = file;
			const reader = new FileReader();
			reader.onload = (event) => {
				logoDataURL = event.target?.result as string;
			};
			reader.readAsDataURL(file);
		}
	}

	function resetLogo() {
		logo = null;
		logoDataURL = null;
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
		<div class="mt-2 flex items-end gap-3">
			{#if logoDataURL}
				<ImageBox
					class="size-24"
					src={logoDataURL}
					alt={m.name_logo({ name: $inputs.name.value })}
				/>
			{/if}
			<div class="flex flex-col gap-2">
				<FileInput
					id="logo"
					variant="secondary"
					accept="image/png, image/jpeg, image/svg+xml"
					onchange={onLogoChange}
				>
					<Button variant="secondary">
						{logoDataURL ? m.change_logo() : m.upload_logo()}
					</Button>
				</FileInput>
				{#if logoDataURL}
					<Button variant="outline" onclick={resetLogo}>{m.remove_logo()}</Button>
				{/if}
			</div>
		</div>
	</div>

	{#if showAdvancedOptions}
		<div class="mt-5 md:col-span-2" transition:slide={{ duration: 200 }}>
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
