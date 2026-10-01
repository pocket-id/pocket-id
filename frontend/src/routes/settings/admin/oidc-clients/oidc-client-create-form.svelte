<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import FormattedMessage from '$lib/components/formatted-message.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Field from '$lib/components/ui/field';
	import * as RadioGroup from '$lib/components/ui/radio-group';
	import { m } from '$lib/paraglide/messages';
	import type { OidcClientCreate } from '$lib/types/oidc.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { callbackUrlSchema, emptyToUndefined } from '$lib/utils/zod-util';
	import { z } from 'zod/v4';
	import OidcCallbackUrlInput from './oidc-callback-url-input.svelte';

	let {
		callback
	}: {
		callback: (client: OidcClientCreate) => Promise<void>;
	} = $props();

	// New clients start with these lifetimes, they can be changed on the details page afterwards
	const DEFAULT_ACCESS_TOKEN_DURATION_MINUTES = 60;
	const DEFAULT_REFRESH_TOKEN_DURATION_MINUTES = 30 * 24 * 60;

	let isLoading = $state(false);
	let showCustomClientId = $state(false);

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
		isPublic: z.boolean()
	});

	const formStore = createForm(formSchema, {
		id: '',
		name: '',
		callbackURLs: [],
		isPublic: false
	});
	const { inputs } = formStore;

	async function onSubmit() {
		const data = formStore.validate();
		if (!data) return;

		// Only the essentials are asked for here, everything else starts with its default and is configured on the details page
		isLoading = true;
		try {
			await callback({
				...data,
				description: '',
				logoutCallbackURLs: [],
				backchannelLogoutURL: '',
				pkceEnabled: data.isPublic,
				requiresReauthentication: false,
				requiresPushedAuthorizationRequests: false,
				skipConsent: false,
				isGroupRestricted: true,
				credentials: { federatedIdentities: [], secrets: [] },
				accessTokenDurationMinutes: DEFAULT_ACCESS_TOKEN_DURATION_MINUTES,
				refreshTokenDurationMinutes: DEFAULT_REFRESH_TOKEN_DURATION_MINUTES
			});
		} catch (e) {
			axiosErrorToast(e);
		} finally {
			isLoading = false;
		}
	}
</script>

{#snippet callbackUrlDescription()}
	<FormattedMessage message={m.callback_url_description} />
{/snippet}

{#snippet clientTypeOption(value: string, title: string, description: string)}
	<Field.Label for="client-type-{value}">
		<Field.Field orientation="horizontal">
			<RadioGroup.Item {value} id="client-type-{value}" />
			<Field.Content>
				<Field.Title>{title}</Field.Title>
				<Field.Description>{description}</Field.Description>
			</Field.Content>
		</Field.Field>
	</Field.Label>
{/snippet}

<form onsubmit={preventDefault(onSubmit)} class="flex flex-col gap-6">
	<div class="grid grid-cols-1 gap-x-3 gap-y-6 md:grid-cols-2">
		<FormInput
			label={m.name()}
			description={m.client_name_description()}
			bind:input={$inputs.name}
		/>
		{#if showCustomClientId}
			<FormInput
				label={m.client_id()}
				placeholder={m.generated()}
				description={m.custom_client_id_description()}
				bind:input={$inputs.id}
			/>
		{/if}
	</div>

	<Field.Set>
		<Field.Legend variant="label">{m.client_type()}</Field.Legend>
		<RadioGroup.Root
			class="grid gap-3 sm:grid-cols-2"
			value={$inputs.isPublic.value ? 'public' : 'confidential'}
			onValueChange={(value) => ($inputs.isPublic.value = value === 'public')}
		>
			{@render clientTypeOption(
				'confidential',
				m.confidential_client(),
				m.confidential_client_description()
			)}
			{@render clientTypeOption('public', m.public_client(), m.public_client_type_description())}
		</RadioGroup.Root>
	</Field.Set>

	<OidcCallbackUrlInput
		label={m.callback_urls()}
		description={callbackUrlDescription}
		addLabel={m.add_callback_url()}
		bind:callbackURLs={$inputs.callbackURLs.value}
		bind:error={$inputs.callbackURLs.error}
	/>

	<div class="flex items-center justify-between gap-3">
		<!-- The client ID can't be changed later, so this is the only place to set a custom one -->
		{#if !showCustomClientId}
			<Button
				variant="ghost"
				size="sm"
				class="text-muted-foreground -ml-2.5"
				onclick={() => (showCustomClientId = true)}
			>
				{m.set_custom_client_id()}
			</Button>
		{/if}
		<Button {isLoading} type="submit" class="ml-auto">{m.create()}</Button>
	</div>
</form>
