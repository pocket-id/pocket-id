<script lang="ts">
	import AdvancedDisclosure from '#lib/components/advanced-disclosure.svelte';
	import DurationInput from '#lib/components/form/duration-input.svelte';
	import FormInput from '#lib/components/form/form-input.svelte';
	import FormattedMessage from '#lib/components/formatted-message.svelte';
	import SettingRow from '#lib/components/setting-row.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import { Switch } from '#lib/components/ui/switch/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type {
		OidcClient,
		OidcClientCreateWithLogo,
		OidcClientLogoPreset
	} from '#lib/types/oidc.type.ts';
	import { cachedOidcClientLogo } from '#lib/utils/cached-image-util.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { preventDefault } from '#lib/utils/event-util.ts';
	import { createForm } from '#lib/utils/form-util.ts';
	import { trackFormChanges } from '#lib/utils/unsaved-changes-util.svelte.ts';
	import { callbackUrlSchema, optionalUrl } from '#lib/utils/zod-util.ts';
	import { z } from 'zod/v4';
	import ChangedMarker from './changed-marker.svelte';
	import ClientTypeTiles from './client-type-tiles.svelte';
	import OidcCallbackUrlInput from './oidc-callback-url-input.svelte';
	import OidcClientLogoPicker from './oidc-client-logo-picker.svelte';

	let {
		callback,
		existingClient
	}: {
		existingClient: OidcClient;
		callback: (client: OidcClientCreateWithLogo) => Promise<void>;
	} = $props();

	// What a new client starts with; anything else counts as changed and keeps Advanced open
	const defaultAccessTokenMinutes = 60;
	const defaultRefreshTokenMinutes = 30 * 24 * 60;

	let logo = $state<File | null | undefined>();
	let darkLogo = $state<File | null | undefined>();
	// What discarding restores the previews to; moves forward whenever a logo is saved
	let savedLogoDataURL = existingClient.hasLogo
		? cachedOidcClientLogo.getUrl(existingClient.id)
		: null;
	let savedDarkLogoDataURL = existingClient.hasDarkLogo
		? cachedOidcClientLogo.getUrl(existingClient.id, false)
		: null;
	let logoDataURL: string | null = $state(savedLogoDataURL);
	let darkLogoDataURL: string | null = $state(savedDarkLogoDataURL);
	const isCIMDClient = existingClient.clientType === 'cimd';

	const durationSchema = z
		.number()
		.min(1, { message: m.token_lifetime_minimum() })
		.max(365 * 24 * 60, { message: m.token_lifetime_maximum() })
		.refine((minutes) => Number.isInteger(minutes), {
			message: m.token_lifetime_whole_minutes()
		});

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
		darkLogoUrl: optionalUrl,
		accessTokenDurationMinutes: durationSchema,
		refreshTokenDurationMinutes: durationSchema
	});

	type FormSchema = typeof formSchema;
	const formStore = createForm<FormSchema>(formSchema, {
		name: existingClient.name,
		description: existingClient.description || '',
		callbackURLs: existingClient.callbackURLs,
		logoutCallbackURLs: existingClient.logoutCallbackURLs,
		backchannelLogoutURL: existingClient.backchannelLogoutURL || '',
		isPublic: existingClient.isPublic,
		pkceEnabled: existingClient.pkceEnabled,
		requiresReauthentication: existingClient.requiresReauthentication,
		requiresPushedAuthorizationRequests: existingClient.requiresPushedAuthorizationRequests,
		skipConsent: existingClient.skipConsent,
		launchURL: existingClient.launchURL || '',
		logoUrl: '',
		darkLogoUrl: '',
		accessTokenDurationMinutes: existingClient.accessTokenDurationMinutes,
		refreshTokenDurationMinutes: existingClient.refreshTokenDurationMinutes
	});
	const { inputs } = formStore;

	const changed = $derived({
		logout: $inputs.logoutCallbackURLs.value.length > 0,
		backchannel: !!$inputs.backchannelLogoutURL.value,
		consent: $inputs.skipConsent.value,
		par: $inputs.requiresPushedAuthorizationRequests.value,
		access: $inputs.accessTokenDurationMinutes.value !== defaultAccessTokenMinutes,
		refresh: $inputs.refreshTokenDurationMinutes.value !== defaultRefreshTokenMinutes
	});
	const changedCount = $derived(Object.values(changed).filter(Boolean).length);
	const pkceRecommended = $derived(
		existingClient.pkceSupported && !$inputs.isPublic.value && !$inputs.pkceEnabled.value
	);

	async function saveClient(data: z.infer<FormSchema>) {
		await callback({
			...data,
			credentials: existingClient.credentials ?? { federatedIdentities: [], secrets: [] },
			logo: $inputs.logoUrl?.value ? undefined : logo,
			logoUrl: $inputs.logoUrl?.value,
			darkLogo: $inputs.darkLogoUrl?.value ? undefined : darkLogo,
			darkLogoUrl: $inputs.darkLogoUrl?.value,
			isGroupRestricted: existingClient.isGroupRestricted
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
		// The uploaded file has been persisted, so it's no longer "pending" for dirty-tracking
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

	// A preset fills both variants, and clears the dark logo when the icon has no dark variant so a stale one isn't kept
	function selectPreset(preset: OidcClientLogoPreset) {
		onLogoChange(preset.logoUrl, true);
		if (preset.darkLogoUrl) {
			onLogoChange(preset.darkLogoUrl, false);
		} else {
			resetLogo(false);
		}
	}

	function resetLogo(light: boolean = true) {
		if (light) {
			logo = null;
			logoDataURL = null;
			formStore.setValue('logoUrl', '');
		} else {
			darkLogo = null;
			darkLogoDataURL = null;
			formStore.setValue('darkLogoUrl', '');
		}
	}

	// A public client always uses PKCE, so switching to Public turns it on
	function setPublic(isPublic: boolean) {
		$inputs.isPublic.value = isPublic;
		if (isPublic) $inputs.pkceEnabled.value = true;
	}
</script>

{#snippet callbackUrlDescription()}
	<FormattedMessage message={m.callback_url_description} />
{/snippet}

{#snippet switchRow(
	key: 'skipConsent' | 'requiresPushedAuthorizationRequests',
	title: string,
	description: string,
	isChanged: boolean
)}
	<SettingRow
		{title}
		{description}
		changed={isChanged}
		onReset={() => ($inputs[key].value = false)}
	>
		{#snippet control(props)}
			<Switch {...props} bind:checked={$inputs[key].value} />
		{/snippet}
	</SettingRow>
{/snippet}

<form onsubmit={preventDefault(onSubmit)} class="flex flex-col gap-10">
	<SettingsSection title={m.application()}>
		<div class="flex flex-col gap-6 pt-2">
			<OidcClientLogoPicker
				clientName={$inputs.name.value}
				{logoDataURL}
				{darkLogoDataURL}
				iconLibrary={$appConfigStore.iconLibrary}
				{onLogoChange}
				onPresetSelect={selectPreset}
				onReset={() => {
					resetLogo(true);
					resetLogo(false);
				}}
			/>
			<div class="grid gap-6 sm:grid-cols-2">
				<FormInput
					label={m.name()}
					description={isCIMDClient ? m.set_by_the_metadata_document() : undefined}
					readonly={isCIMDClient}
					bind:input={$inputs.name}
				/>
				<FormInput
					label={m.description()}
					optional
					description={m.count_of_max({
						count: $inputs.description.value.length,
						max: 150
					})}
					bind:input={$inputs.description}
				/>
			</div>
			<FormInput
				label={m.launch_url()}
				optional
				description={m.opened_when_someone_launches_name_from_my_apps()}
				placeholder="https://"
				type="url"
				inputClass="font-mono"
				bind:input={$inputs.launchURL}
			/>
			<ClientTypeTiles
				bind:isPublic={() => $inputs.isPublic.value, setPublic}
				disabled={isCIMDClient}
			/>
		</div>
	</SettingsSection>

	<SettingsSection title={m.redirects()}>
		<div class="pt-2">
			<OidcCallbackUrlInput
				label={m.callback_urls()}
				description={callbackUrlDescription}
				placeholder="https://app.example.com/auth/callback"
				emptyText={m.no_callback_urls()}
				testIdPrefix="callback-url"
				readonly={isCIMDClient}
				bind:urls={$inputs.callbackURLs.value}
				error={$inputs.callbackURLs.error}
			/>
		</div>
	</SettingsSection>

	<SettingsSection title={m.security()}>
		<div>
			<SettingRow
				title={m.pkce()}
				description={m.pkce_row_description()}
				consequence={$inputs.isPublic.value
					? m.public_clients_always_use_pkce()
					: pkceRecommended
						? m.name_supports_pkce({ name: existingClient.name })
						: undefined}
			>
				{#snippet control(props)}
					<Switch
						{...props}
						disabled={isCIMDClient || $inputs.isPublic.value}
						bind:checked={$inputs.pkceEnabled.value}
					/>
				{/snippet}
			</SettingRow>
			<SettingRow
				title={m.requires_reauthentication()}
				description={m.requires_reauthentication_row_description()}
			>
				{#snippet control(props)}
					<Switch {...props} bind:checked={$inputs.requiresReauthentication.value} />
				{/snippet}
			</SettingRow>
		</div>
	</SettingsSection>

	<AdvancedDisclosure summary={m.oidc_client_advanced_summary()} {changedCount}>
		<SettingsSection title={m.sign_out_section()}>
			<div class="flex flex-col gap-6 pt-2">
				<OidcCallbackUrlInput
					label={m.logout_callback_urls()}
					description={callbackUrlDescription}
					placeholder="https://app.example.com/logout"
					emptyText={m.no_logout_callback_urls()}
					testIdPrefix="logout-callback-url"
					readonly={isCIMDClient}
					changed={changed.logout}
					onReset={isCIMDClient ? undefined : () => ($inputs.logoutCallbackURLs.value = [])}
					bind:urls={$inputs.logoutCallbackURLs.value}
					error={$inputs.logoutCallbackURLs.error}
				/>
				<div class="relative min-w-0">
					<FormInput
						label={m.backchannel_logout_url()}
						optional
						description={m.backchannel_logout_url_helper()}
						type="url"
						inputClass="font-mono"
						readonly={isCIMDClient}
						changed={changed.backchannel}
						bind:input={$inputs.backchannelLogoutURL}
					/>
					{#if changed.backchannel && !isCIMDClient}
						<ChangedMarker
							label={m.backchannel_logout_url()}
							onReset={() => ($inputs.backchannelLogoutURL.value = '')}
						/>
					{/if}
				</div>
			</div>
		</SettingsSection>

		<SettingsSection title={m.authorization()}>
			<div>
				{@render switchRow(
					'skipConsent',
					m.skip_consent(),
					m.skip_consent_row_description({ name: existingClient.name }),
					changed.consent
				)}
				{@render switchRow(
					'requiresPushedAuthorizationRequests',
					m.requires_pushed_authorization_requests(),
					m.requires_par_row_description(),
					changed.par
				)}
			</div>
		</SettingsSection>

		<SettingsSection
			title={m.token_lifetimes()}
			description={m.token_lifetimes_section_description()}
		>
			<div class="grid gap-6 pt-2 sm:grid-cols-2">
				<div class="relative min-w-0">
					<DurationInput
						id="access-token-lifetime"
						label={m.access_token_lifetime()}
						changed={changed.access}
						bind:input={$inputs.accessTokenDurationMinutes}
					/>
					{#if changed.access}
						<ChangedMarker
							label={m.access_token_lifetime()}
							onReset={() => ($inputs.accessTokenDurationMinutes.value = defaultAccessTokenMinutes)}
						/>
					{/if}
				</div>
				<div class="relative min-w-0">
					<DurationInput
						id="refresh-token-lifetime"
						label={m.refresh_token_inactivity_timeout()}
						description={m.refresh_token_lifetime_helper({ name: existingClient.name })}
						changed={changed.refresh}
						bind:input={$inputs.refreshTokenDurationMinutes}
					/>
					{#if changed.refresh}
						<ChangedMarker
							label={m.refresh_token_inactivity_timeout()}
							onReset={() =>
								($inputs.refreshTokenDurationMinutes.value = defaultRefreshTokenMinutes)}
						/>
					{/if}
				</div>
			</div>
		</SettingsSection>
	</AdvancedDisclosure>
</form>
