<script lang="ts">
	import { goto } from '$app/navigation';
	import FormInput from '#lib/components/form/form-input.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Sheet from '#lib/components/ui/sheet/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import OIDCService from '#lib/services/oidc-service.ts';
	import clientSecretStore from '#lib/stores/client-secret-store.ts';
	import { encodeClientIdParam } from '#lib/utils/client-id-util.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { preventDefault } from '#lib/utils/event-util.ts';
	import { createForm } from '#lib/utils/form-util.ts';
	import { callbackUrlSchema, emptyToUndefined } from '#lib/utils/zod-util.ts';
	import { LucideChevronDown, LucideInfo } from '@lucide/svelte';
	import { untrack } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod/v4';
	import ClientTypeTiles from './client-type-tiles.svelte';

	let { open = $bindable() }: { open: boolean } = $props();

	const oidcService = new OIDCService();
	const formId = $props.id();

	let isLoading = $state(false);
	let showCustomClientId = $state(false);

	const formSchema = z.object({
		name: z.string().min(2).max(50),
		isPublic: z.boolean(),
		callbackURL: emptyToUndefined(callbackUrlSchema),
		id: emptyToUndefined(
			z
				.string()
				.min(2)
				.max(128)
				.regex(/^[a-zA-Z0-9_-]+$/, { message: m.invalid_client_id() })
		)
	});

	const formStore = createForm(formSchema, { name: '', isPublic: false, callbackURL: '', id: '' });
	const { inputs } = formStore;

	// A closed sheet starts empty the next time it opens
	$effect(() => {
		if (open) return;
		untrack(() => {
			formStore.reset();
			showCustomClientId = false;
		});
	});

	async function onSubmit() {
		const data = formStore.validate();
		if (!data) return;

		// Only the essentials are asked for here, everything else starts with its default and is configured on the details page
		isLoading = true;
		try {
			clientSecretStore.clear();
			const client = await oidcService.createClient({
				id: data.id,
				name: data.name,
				isPublic: data.isPublic,
				callbackURLs: data.callbackURL ? [data.callbackURL] : [],
				description: '',
				logoutCallbackURLs: [],
				backchannelLogoutURL: '',
				pkceEnabled: data.isPublic,
				requiresReauthentication: false,
				requiresPushedAuthorizationRequests: false,
				skipConsent: false,
				isGroupRestricted: true,
				credentials: { federatedIdentities: [], secrets: [] },
				accessTokenDurationMinutes: 60,
				refreshTokenDurationMinutes: 30 * 24 * 60
			});
			if (client.createdSecret) {
				clientSecretStore.setAutoCreated(client.createdSecret.id, client.createdSecret.secret);
			}
			open = false;
			goto(`/settings/admin/oidc-clients/${encodeClientIdParam(client.id)}`);
			toast.success(m.name_created({ name: client.name }));
		} catch (e) {
			axiosErrorToast(e);
		} finally {
			isLoading = false;
		}
	}
</script>

<Sheet.Root bind:open>
	<Sheet.Content title={m.add_oidc_client()}>
		<form id={formId} onsubmit={preventDefault(onSubmit)} class="flex flex-col gap-6">
			<FormInput
				label={m.name()}
				placeholder="Planka"
				description={m.client_name_helper()}
				bind:input={$inputs.name}
			/>
			<ClientTypeTiles bind:isPublic={$inputs.isPublic.value} />
			<FormInput
				label={m.callback_url()}
				optional
				placeholder="https://boards.example.com/oidc/callback"
				description={m.create_callback_url_helper()}
				inputClass="font-mono"
				bind:input={$inputs.callbackURL}
			/>
			<div class="flex flex-col gap-4">
				<button
					type="button"
					aria-expanded={showCustomClientId}
					class="text-control text-fg-muted hover:text-fg inline-flex w-fit items-center gap-1 rounded-xs"
					onclick={() => (showCustomClientId = !showCustomClientId)}
				>
					<LucideChevronDown
						class="size-3.5 transition-transform duration-(--dur-base) {showCustomClientId
							? ''
							: '-rotate-90'}"
					/>
					{m.set_a_custom_client_id()}
				</button>
				{#if showCustomClientId}
					<FormInput
						class="transition-motion duration-(--dur-base) starting:-translate-y-(--rise) starting:opacity-0"
						label={m.client_id()}
						optional
						placeholder={m.generated()}
						description={m.custom_client_id_helper()}
						inputClass="font-mono"
						bind:input={$inputs.id}
					/>
				{/if}
			</div>
			<p class="text-caption text-fg-muted flex items-start gap-1.5">
				<LucideInfo class="mt-[3px] size-3 shrink-0" />
				{m.new_clients_start_with_no_allowed_groups()}
			</p>
		</form>
		{#snippet footer()}
			<Button variant="ghost" onclick={() => (open = false)}>{m.cancel()}</Button>
			<Button type="submit" form={formId} {isLoading}>{m.create_client()}</Button>
		{/snippet}
	</Sheet.Content>
</Sheet.Root>
