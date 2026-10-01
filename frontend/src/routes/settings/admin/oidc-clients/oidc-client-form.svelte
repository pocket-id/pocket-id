<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import FormattedMessage from '$lib/components/formatted-message.svelte';
	import * as Card from '$lib/components/ui/card';
	import * as Field from '$lib/components/ui/field';
	import { Switch } from '$lib/components/ui/switch';
	import * as Tabs from '$lib/components/ui/tabs';
	import { m } from '$lib/paraglide/messages';
	import type { OidcClient, OidcClientCreateWithLogo } from '$lib/types/oidc.type';
	import { cachedOidcClientLogo } from '$lib/utils/cached-image-util';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm, type FormInput as FormInputState } from '$lib/utils/form-util';
	import { trackFormChanges } from '$lib/utils/unsaved-changes-util.svelte';
	import { callbackUrlSchema, optionalUrl } from '$lib/utils/zod-util';
	import { LucideMoon, LucideSun } from '@lucide/svelte';
	import { z } from 'zod/v4';
	import OidcCallbackUrlInput from './oidc-callback-url-input.svelte';
	import OidcClientImageInput from './oidc-client-image-input.svelte';

	let {
		callback,
		existingClient
	}: {
		existingClient: OidcClient;
		callback: (client: OidcClientCreateWithLogo) => Promise<void>;
	} = $props();
	let logo = $state<File | null | undefined>();
	let darkLogo = $state<File | null | undefined>();
	// What discarding restores the previews to; moves forward whenever a logo is saved.
	let savedLogoDataURL = existingClient.hasLogo
		? cachedOidcClientLogo.getUrl(existingClient.id)
		: null;
	let savedDarkLogoDataURL = existingClient.hasDarkLogo
		? cachedOidcClientLogo.getUrl(existingClient.id, false)
		: null;
	let logoDataURL: string | null = $state(savedLogoDataURL);
	let darkLogoDataURL: string | null = $state(savedDarkLogoDataURL);
	const isCIMDClient = $derived(existingClient.clientType === 'cimd');

	const client = {
		name: existingClient.name || '',
		description: existingClient.description || '',
		callbackURLs: existingClient.callbackURLs || [],
		logoutCallbackURLs: existingClient.logoutCallbackURLs || [],
		backchannelLogoutURL: existingClient.backchannelLogoutURL || '',
		isPublic: existingClient.isPublic || false,
		pkceEnabled: existingClient.pkceEnabled || false,
		requiresReauthentication: existingClient.requiresReauthentication || false,
		requiresPushedAuthorizationRequests:
			existingClient.requiresPushedAuthorizationRequests || false,
		skipConsent: existingClient.skipConsent || false,
		launchURL: existingClient.launchURL || '',
		logoUrl: '',
		darkLogoUrl: '',
		pkceSupported: existingClient.pkceSupported || false
	};

	const formSchema = z.object({
		name: z.string().min(2).max(50),
		description: z.string().max(150),
		callbackURLs: z.array(callbackUrlSchema).default([]),
		logoutCallbackURLs: z.array(callbackUrlSchema).default([]),
		backchannelLogoutURL: z.url().or(z.literal('')),
		isPublic: z.boolean(),
		pkceEnabled: z.boolean(),
		requiresReauthentication: z.boolean(),
		requiresPushedAuthorizationRequests: z.boolean(),
		skipConsent: z.boolean(),
		launchURL: optionalUrl,
		logoUrl: optionalUrl,
		darkLogoUrl: optionalUrl
	});

	type FormSchema = typeof formSchema;
	const formStore = createForm<FormSchema>(formSchema, client);
	const { inputs } = formStore;

	const pkcePromptNeeded = $derived(!$inputs.pkceEnabled.value && client.pkceSupported);

	async function saveClient(data: z.infer<FormSchema>) {
		await callback({
			...data,
			credentials: existingClient.credentials ?? { federatedIdentities: [], secrets: [] },
			logo: $inputs.logoUrl?.value ? undefined : logo,
			logoUrl: $inputs.logoUrl?.value,
			darkLogo: $inputs.darkLogoUrl?.value ? undefined : darkLogo,
			darkLogoUrl: $inputs.darkLogoUrl?.value,
			isGroupRestricted: existingClient.isGroupRestricted,
			// The token lifetimes are edited in their own card, but the current values are sent along because the backend falls back to the defaults for missing ones
			accessTokenDurationMinutes: existingClient.accessTokenDurationMinutes,
			refreshTokenDurationMinutes: existingClient.refreshTokenDurationMinutes
		});

		const hasLogo = logo != null || !!$inputs.logoUrl?.value;
		const hasDarkLogo = darkLogo != null || !!$inputs.darkLogoUrl?.value;
		if (hasLogo) {
			logoDataURL = cachedOidcClientLogo.getUrl(existingClient.id);
		}
		if (hasDarkLogo) {
			darkLogoDataURL = cachedOidcClientLogo.getUrl(existingClient.id, false);
		}
		savedLogoDataURL = logoDataURL;
		savedDarkLogoDataURL = darkLogoDataURL;
		// The uploaded file has been persisted, so it's no longer "pending" for dirty-tracking.
		logo = undefined;
		darkLogo = undefined;
	}

	// Submitting with the Enter key saves right away instead of going through the unsaved-changes bar
	async function onSubmit() {
		const data = formStore.validate();
		if (!data) return;
		await saveClient(data).catch(axiosErrorToast);
	}

	function discardLogoChanges() {
		logo = undefined;
		darkLogo = undefined;
		logoDataURL = savedLogoDataURL;
		darkLogoDataURL = savedDarkLogoDataURL;
	}

	trackFormChanges(() => formStore, saveClient, {
		dirty: () => logo !== undefined || darkLogo !== undefined,
		discard: discardLogoChanges
	});

	function onLogoChange(input: File | string | null, light: boolean = true) {
		if (input == null) return;

		const logoUrlKey = light ? 'logoUrl' : 'darkLogoUrl';

		if (typeof input === 'string') {
			if (light) {
				logo = null;
				logoDataURL = input || null;
			} else {
				darkLogo = null;
				darkLogoDataURL = input || null;
			}
			formStore.setValue(logoUrlKey, input);
		} else {
			if (light) {
				logo = input;
				logoDataURL = URL.createObjectURL(input);
			} else {
				darkLogo = input;
				darkLogoDataURL = URL.createObjectURL(input);
			}
			formStore.setValue(logoUrlKey, '');
		}
	}

	function resetLogo(light: boolean = true) {
		if (light) {
			logo = null;
			logoDataURL = null;
			if ($inputs.logoUrl) {
				$inputs.logoUrl.value = '';
			}
		} else {
			darkLogo = null;
			darkLogoDataURL = null;
			if ($inputs.darkLogoUrl) {
				$inputs.darkLogoUrl.value = '';
			}
		}
	}
</script>

{#snippet callbackUrlDescription()}
	<FormattedMessage message={m.callback_url_description} />
{/snippet}

{#snippet logoutCallbackUrlDescription()}
	<FormattedMessage message={m.logout_callback_url_description} />
{/snippet}

{#snippet switchField(
	id: string,
	label: string,
	description: string,
	input: FormInputState<boolean>,
	disabled: boolean = false,
	onCheckedChange?: (checked: boolean) => void
)}
	<Field.Field orientation="horizontal" data-disabled={disabled}>
		<Field.Content>
			<Field.Label for={id}>{label}</Field.Label>
			<Field.Description>{description}</Field.Description>
		</Field.Content>
		<Switch {id} {disabled} {onCheckedChange} bind:checked={input.value} />
	</Field.Field>
{/snippet}

{#snippet publicClientField()}
	{@render switchField(
		'public-client',
		m.public_client(),
		m.public_clients_description(),
		$inputs.isPublic,
		isCIMDClient,
		(checked) => {
			if (checked) {
				$inputs.pkceEnabled.value = true;
			}
		}
	)}
{/snippet}

{#snippet pkceField()}
	<div
		class="rounded-lg transition-all duration-200"
		class:[&_[data-switch-root]]:ring-2={pkcePromptNeeded}
		class:[&_[data-switch-root]]:ring-blue-500={pkcePromptNeeded}
	>
		{@render switchField(
			'pkce',
			m.pkce(),
			m.proof_key_code_exchange_is_a_security_feature_to_prevent_csrf_and_authorization_code_interception_attacks(),
			$inputs.pkceEnabled,
			isCIMDClient || $inputs.isPublic.value
		)}
	</div>
{/snippet}

{#snippet reauthenticationField()}
	{@render switchField(
		'requires-reauthentication',
		m.requires_reauthentication(),
		m.requires_users_to_authenticate_again_on_each_authorization(),
		$inputs.requiresReauthentication
	)}
{/snippet}

{#snippet skipConsentField()}
	{@render switchField(
		'skip-consent',
		m.skip_consent(),
		m.skip_consent_description(),
		$inputs.skipConsent
	)}
{/snippet}

{#snippet parField()}
	{@render switchField(
		'requires-par',
		m.requires_pushed_authorization_requests(),
		m.requires_pushed_authorization_requests_description(),
		$inputs.requiresPushedAuthorizationRequests
	)}
{/snippet}

{#snippet callbackUrlsInput()}
	<OidcCallbackUrlInput
		label={m.callback_urls()}
		description={callbackUrlDescription}
		addLabel={m.add_callback_url()}
		class="w-full"
		bind:callbackURLs={$inputs.callbackURLs.value}
		bind:error={$inputs.callbackURLs.error}
		disabled={isCIMDClient}
	/>
{/snippet}

{#snippet logoutCallbackUrlsInput()}
	<OidcCallbackUrlInput
		label={m.logout_callback_urls()}
		description={logoutCallbackUrlDescription}
		addLabel={m.add_logout_url()}
		class="w-full"
		bind:callbackURLs={$inputs.logoutCallbackURLs.value}
		bind:error={$inputs.logoutCallbackURLs.error}
		disabled={isCIMDClient}
	/>
{/snippet}

{#snippet backchannelLogoutUrlInput()}
	<FormInput
		label={m.backchannel_logout_url()}
		description={m.backchannel_logout_url_description()}
		class="w-full"
		type="url"
		bind:input={$inputs.backchannelLogoutURL}
		disabled={isCIMDClient}
	/>
{/snippet}

{#snippet logoTabTriggers()}
	<Tabs.List class="grid h-8 grid-cols-2">
		<Tabs.Trigger value="light-logo" class="px-2.5" aria-label={m.light()}>
			<LucideSun class="size-3.5" />
		</Tabs.Trigger>
		<Tabs.Trigger value="dark-logo" class="px-2.5" aria-label={m.dark()}>
			<LucideMoon class="size-3.5" />
		</Tabs.Trigger>
	</Tabs.List>
{/snippet}

{#snippet logoInput()}
	<Tabs.Root value="light-logo">
		<Tabs.Content value="light-logo">
			<OidcClientImageInput
				{logoDataURL}
				resetLogo={() => resetLogo(true)}
				clientName={$inputs.name.value}
				light={true}
				onLogoChange={(input) => onLogoChange(input, true)}
				tabTriggers={logoTabTriggers}
			/>
		</Tabs.Content>
		<Tabs.Content value="dark-logo">
			<OidcClientImageInput
				light={false}
				logoDataURL={darkLogoDataURL}
				resetLogo={() => resetLogo(false)}
				clientName={$inputs.name.value}
				onLogoChange={(input) => onLogoChange(input, false)}
				tabTriggers={logoTabTriggers}
			/>
		</Tabs.Content>
	</Tabs.Root>
{/snippet}

<!-- The form is split into cards so that related settings are grouped together -->
<form onsubmit={preventDefault(onSubmit)} class="flex flex-col gap-6">
	<Card.Root>
		<Card.Header>
			<Card.Title>{m.application()}</Card.Title>
			<Card.Description>{m.oidc_client_application_description()}</Card.Description>
		</Card.Header>
		<Card.Content class="flex flex-col gap-7">
			<!-- The logo, name and description form one block because together they are what users see on the consent screen -->
			<div class="flex flex-col gap-7 sm:flex-row sm:gap-6">
				<div class="shrink-0">
					{@render logoInput()}
				</div>
				<div class="flex flex-1 flex-col gap-6">
					<FormInput
						label={m.name()}
						class="w-full"
						bind:input={$inputs.name}
						disabled={isCIMDClient}
					/>
					<FormInput
						label={m.client_description()}
						class="w-full"
						bind:input={$inputs.description}
					/>
				</div>
			</div>
			<FormInput
				label={m.client_launch_url()}
				description={m.client_launch_url_description()}
				class="w-full"
				type="url"
				bind:input={$inputs.launchURL}
			/>
		</Card.Content>
	</Card.Root>

	<Card.Root>
		<Card.Header>
			<Card.Title>{m.redirects()}</Card.Title>
			<Card.Description>{m.oidc_client_redirects_description()}</Card.Description>
		</Card.Header>
		<Card.Content class="flex flex-col gap-7">
			{@render callbackUrlsInput()}
			{@render logoutCallbackUrlsInput()}
			{@render backchannelLogoutUrlInput()}
		</Card.Content>
	</Card.Root>

	<Card.Root>
		<Card.Header>
			<Card.Title>{m.security()}</Card.Title>
			<Card.Description>{m.oidc_client_security_description()}</Card.Description>
		</Card.Header>
		<Card.Content>
			<Field.Group class="gap-6">
				{@render publicClientField()}
				<Field.Separator />
				{@render pkceField()}
				<Field.Separator />
				{@render reauthenticationField()}
				<Field.Separator />
				{@render skipConsentField()}
				<Field.Separator />
				{@render parField()}
			</Field.Group>
		</Card.Content>
	</Card.Root>
</form>
