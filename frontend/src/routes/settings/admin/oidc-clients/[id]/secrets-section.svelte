<script lang="ts">
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import { Badge } from '#lib/components/ui/badge/index.ts';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Dialog from '#lib/components/ui/dialog/index.ts';
	import * as Field from '#lib/components/ui/field/index.ts';
	import * as Select from '#lib/components/ui/select/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { getLocale } from '#lib/paraglide/runtime.js';
	import OidcService from '#lib/services/oidc-service.ts';
	import clientSecretStore, { autoCreatedSecretId } from '#lib/stores/client-secret-store.ts';
	import type { OidcClient, OidcClientSecret } from '#lib/types/oidc.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { LucidePlus, LucideTrash2 } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import SecretReveal from './secret-reveal.svelte';

	let {
		client,
		secrets = $bindable()
	}: {
		client: OidcClient;
		secrets: OidcClientSecret[];
	} = $props();

	const oidcService = new OidcService();

	// Mirrors model.MaxOidcClientSecrets on the backend, which rejects any secret above this count
	const maxSecrets = 20;

	// A secret can only be given an expiration when it is created, so this is not bound to any existing secret
	const expirationOptions = [
		{ value: 'never', label: m.no_expiration() },
		...[30, 90, 180, 360, 720].map((days) => ({
			value: String(days),
			label: m.days_count({ count: String(days) })
		}))
	];

	let dialogOpen = $state(false);
	let expiration = $state('never');
	let isCreating = $state(false);
	let revealRef = $state<HTMLElement>();
	// Set only by a successful generate, so cancelling the dialog later returns focus to its trigger as usual
	let focusReveal = false;

	const limitReached = $derived(secrets.length >= maxSecrets);
	// The newest secret generated during this visit; the one created with the client shows on the General tab instead
	const revealed = $derived(
		secrets
			.filter((secret) => secret.id !== $autoCreatedSecretId)
			.map((secret) => $clientSecretStore[secret.id])
			.find(Boolean)
	);

	const formatDate = (date: string) =>
		new Date(date).toLocaleDateString(getLocale(), { dateStyle: 'medium' });

	function expirationDate() {
		if (expiration === 'never') return null;
		const millisecondsPerDay = 24 * 60 * 60 * 1000;
		return new Date(Date.now() + Number(expiration) * millisecondsPerDay);
	}

	function openGenerate() {
		expiration = 'never';
		dialogOpen = true;
	}

	async function createSecret() {
		isCreating = true;
		try {
			const created = await oidcService.createClientSecret(client.id, expirationDate());
			// The value is kept in the store, so it survives switching between tabs but not leaving the page
			clientSecretStore.set(created.id, created.secret);
			secrets = [created, ...secrets];
			focusReveal = true;
			dialogOpen = false;
			toast.success(m.new_client_secret_generated());
		} catch (e) {
			axiosErrorToast(e);
		} finally {
			isCreating = false;
		}
	}

	function deleteSecret(secret: OidcClientSecret) {
		openConfirmDialog({
			title: m.delete_client_secret_question(),
			message: m.delete_client_secret_consequence(),
			confirm: {
				label: m.delete_secret(),
				destructive: true,
				action: async () => {
					try {
						await oidcService.deleteClientSecret(client.id, secret.id);
						clientSecretStore.remove(secret.id);
						secrets = secrets.filter((s) => s.id !== secret.id);
						toast.success(m.client_secret_deleted());
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}
</script>

<SettingsSection
	title={m.client_secrets()}
	description={client.isPublic
		? undefined
		: limitReached
			? m.client_secrets_limit_reached()
			: m.client_secrets_section_description()}
>
	<!-- An empty list offers the action in its row, so the header shows it only once secrets exist -->
	{#snippet action()}
		{#if !client.isPublic && secrets.length > 0}
			<Button variant="secondary" size="sm" disabled={limitReached} onclick={openGenerate}>
				<LucidePlus />
				{m.generate_secret_ellipsis()}
			</Button>
		{/if}
	{/snippet}
	<div class="mt-2 flex flex-col gap-4">
		{#if revealed}
			<div bind:this={revealRef}>
				<SecretReveal label={m.new_client_secret()} value={revealed} testId="new-client-secret" />
			</div>
		{/if}
		<ul class="flex flex-col rounded-md border">
			{#if client.isPublic}
				<li class="text-control text-fg-muted flex min-h-11 items-center px-4">
					{m.public_clients_dont_have_client_secrets()}
				</li>
			{:else}
				{#each secrets as secret (secret.id)}
					<li
						class="flex min-h-11 items-center gap-3 border-b py-2 ps-4 pe-2 last:border-b-0"
						data-testid="client-secret-row"
					>
						<div class="flex min-w-0 flex-1 flex-col">
							<span class="text-mono-s font-mono" data-testid="client-secret">
								{secret.prefix}<span aria-hidden="true">••••••••</span>
							</span>
							<span class="text-caption text-fg-muted flex flex-wrap items-center gap-x-1.5">
								{m.created_date({ date: formatDate(secret.createdAt) })} ·
								{#if !secret.isActive}
									<Badge variant="warning" class="text-caption text-fg-muted before:size-1.5">
										{m.expired()}
									</Badge>
								{:else if secret.expiresAt}
									{m.expires_date({ date: formatDate(secret.expiresAt) })}
								{:else}
									{m.never_expires()}
								{/if}
							</span>
						</div>
						<Button
							variant="ghost"
							size="icon-xs"
							aria-label={m.delete_secret_prefix({ prefix: secret.prefix })}
							onclick={() => deleteSecret(secret)}
						>
							<LucideTrash2 />
						</Button>
					</li>
				{:else}
					<li class="flex min-h-11 flex-wrap items-center justify-between gap-2 ps-4 pe-2">
						<span class="text-control text-fg-muted min-w-0">
							{m.name_has_no_client_secrets_yet({ name: client.name })}
						</span>
						<Button variant="ghost" size="sm" onclick={openGenerate}>
							<LucidePlus />
							{m.generate_secret_ellipsis()}
						</Button>
					</li>
				{/each}
			{/if}
		</ul>
	</div>
</SettingsSection>

<Dialog.Root bind:open={dialogOpen}>
	<!-- The new secret's only button is its Copy button, which takes focus so it can be copied right away -->
	<Dialog.Content
		class="max-w-(--dialog-sm)"
		onCloseAutoFocus={(e) => {
			if (!focusReveal) return;
			focusReveal = false;
			e.preventDefault();
			revealRef?.querySelector('button')?.focus();
		}}
	>
		<Dialog.Header>
			<Dialog.Title>{m.generate_a_client_secret()}</Dialog.Title>
		</Dialog.Header>
		<Field.Field>
			<Field.Label for="secret-expiration">{m.expires_after()}</Field.Label>
			<Select.Root type="single" bind:value={expiration}>
				<Select.Trigger id="secret-expiration" class="w-full">
					{expirationOptions.find((option) => option.value === expiration)?.label}
				</Select.Trigger>
				<Select.Content>
					{#each expirationOptions as option (option.value)}
						<Select.Item value={option.value}>{option.label}</Select.Item>
					{/each}
				</Select.Content>
			</Select.Root>
		</Field.Field>
		<Dialog.Footer>
			<Button variant="ghost" onclick={() => (dialogOpen = false)}>{m.cancel()}</Button>
			<Button isLoading={isCreating} onclick={createSecret}>{m.generate_secret()}</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
