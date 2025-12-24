<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import SwitchWithLabel from '$lib/components/form/switch-with-label.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Tabs from '$lib/components/ui/tabs';
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
	import { LucideChevronDown, LucideMoon, LucideSun } from '@lucide/svelte';
	import { slide } from 'svelte/transition';
	import { z } from 'zod/v4';
	import FederatedIdentitiesInput from './federated-identities-input.svelte';
	import OidcCallbackUrlInput from './oidc-callback-url-input.svelte';
	import OidcClientImageInput from './oidc-client-image-input.svelte';

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
	let darkLogo = $state<File | null | undefined>();
	let logoDataURL: string | null = $state(
		existingClient?.hasLogo ? cachedOidcClientLogo.getUrl(existingClient!.id) : null
	);
	let darkLogoDataURL: string | null = $state(
		existingClient?.hasDarkLogo ? cachedOidcClientLogo.getUrl(existingClient!.id, false) : null
	);

	const client = {
		id: '',
		name: existingClient?.name || '',
		callbackURLs: existingClient?.callbackURLs || [],
		logoutCallbackURLs: existingClient?.logoutCallbackURLs || [],
		isPublic: existingClient?.isPublic || false,
		pkceEnabled: existingClient?.pkceEnabled || false,
		requiresReauthentication: existingClient?.requiresReauthentication || false,
		launchURL: existingClient?.launchURL || '',
		credentials: {
			federatedIdentities: existingClient?.credentials?.federatedIdentities || []
		},
		logoUrl: '',
		darkLogoUrl: ''
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
		darkLogoUrl: optionalUrl,
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
			logo: $inputs.logoUrl?.value ? undefined : logo,
			logoUrl: $inputs.logoUrl?.value,
			darkLogo: $inputs.darkLogoUrl?.value ? undefined : darkLogo,
			darkLogoUrl: $inputs.darkLogoUrl?.value,
			isGroupRestricted: existingClient?.isGroupRestricted ?? true
		});

		const hasLogo = logo != null || !!$inputs.logoUrl?.value;
		const hasDarkLogo = darkLogo != null || !!$inputs.darkLogoUrl?.value;
		if (success && existingClient) {
			if (hasLogo) {
				logoDataURL = cachedOidcClientLogo.getUrl(existingClient.id);
			}
			if (hasDarkLogo) {
				darkLogoDataURL = cachedOidcClientLogo.getUrl(existingClient.id, false);
			}
		}

		if (success && !existingClient) form.reset();
		isLoading = false;
	}

	function onLogoChange(input: File | string | null, light: boolean = true) {
		if (input == null) return;

		const logoUrlInput = light ? $inputs.logoUrl : $inputs.darkLogoUrl;

		if (typeof input === 'string') {
			if (light) {
				logo = null;
				logoDataURL = input || null;
			} else {
				darkLogo = null;
				darkLogoDataURL = input || null;
			}
			logoUrlInput!.value = input;
		} else {
			if (light) {
				logo = input;
				logoDataURL = URL.createObjectURL(input);
			} else {
				darkLogo = input;
				darkLogoDataURL = URL.createObjectURL(input);
			}
			logoUrlInput && (logoUrlInput.value = '');
		}
	}

	function resetLogo(light: boolean = true) {
		if (light) {
			logo = null;
			logoDataURL = null;
			$inputs.logoUrl && ($inputs.logoUrl.value = '');
		} else {
			darkLogo = null;
			darkLogoDataURL = null;
			$inputs.darkLogoUrl && ($inputs.darkLogoUrl.value = '');
		}
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
			onCheckedChange={(v) => {
				if (v) {
					$inputs.pkceEnabled.value = true;
				}
			}}
			bind:checked={$inputs.isPublic.value}
		/>
		<SwitchWithLabel
			id="pkce"
			label={m.pkce()}
			description={m.public_key_code_exchange_is_a_security_feature_to_prevent_csrf_and_authorization_code_interception_attacks()}
			disabled={$inputs.isPublic.value}
			bind:checked={$inputs.pkceEnabled.value}
		/>
		<SwitchWithLabel
			id="requires-reauthentication"
			label={m.requires_reauthentication()}
			description={m.requires_users_to_authenticate_again_on_each_authorization()}
			bind:checked={$inputs.requiresReauthentication.value}
		/>
	</div>
	<div class="mt-7 w-full md:w-1/2">
		<Tabs.Root value="light-logo">
			<Tabs.Content value="light-logo">
				<OidcClientImageInput
					{logoDataURL}
					resetLogo={() => resetLogo(true)}
					clientName={$inputs.name.value}
					light={true}
					onLogoChange={(input) => onLogoChange(input, true)}
				>
					{#snippet tabTriggers()}
						<Tabs.List class="grid h-9 w-full grid-cols-2">
							<Tabs.Trigger value="light-logo" class="px-3">
								<LucideSun class="size-4" />
							</Tabs.Trigger>
							<Tabs.Trigger value="dark-logo" class="px-3">
								<LucideMoon class="size-4" />
							</Tabs.Trigger>
						</Tabs.List>
					{/snippet}
				</OidcClientImageInput>
			</Tabs.Content>
			<Tabs.Content value="dark-logo">
				<OidcClientImageInput
					light={false}
					logoDataURL={darkLogoDataURL}
					resetLogo={() => resetLogo(false)}
					clientName={$inputs.name.value}
					onLogoChange={(input) => onLogoChange(input, false)}
				>
					{#snippet tabTriggers()}
						<Tabs.List class="grid h-9 w-full grid-cols-2">
							<Tabs.Trigger value="light-logo" class="px-3">
								<LucideSun class="size-4" />
							</Tabs.Trigger>
							<Tabs.Trigger value="dark-logo" class="px-3">
								<LucideMoon class="size-4" />
							</Tabs.Trigger>
						</Tabs.List>
					{/snippet}
				</OidcClientImageInput>
			</Tabs.Content>
		</Tabs.Root>
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
