<script lang="ts">
	import { openConfirmDialog } from '$lib/components/confirm-dialog';
	import CopyToClipboard from '$lib/components/copy-to-clipboard.svelte';
	import { Badge } from '$lib/components/ui/badge';
	import { Button, buttonVariants } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import * as Select from '$lib/components/ui/select';
	import * as Table from '$lib/components/ui/table';
	import { m } from '$lib/paraglide/messages';
	import OidcService from '$lib/services/oidc-service';
	import clientSecretStore from '$lib/stores/client-secret-store';
	import type { OidcClient, OidcClientSecret } from '$lib/types/oidc.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideEllipsis, LucidePlus, LucideTrash2 } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';

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
		{ value: '30', label: m.days_count({ count: '30' }) },
		{ value: '90', label: m.days_count({ count: '90' }) },
		{ value: '180', label: m.days_count({ count: '180' }) },
		{ value: '360', label: m.days_count({ count: '360' }) },
		{ value: '720', label: m.days_count({ count: '720' }) }
	];

	let expiration = $state('never');
	let isCreating = $state(false);

	const selectedExpirationLabel = $derived(
		expirationOptions.find((option) => option.value === expiration)!.label
	);
	const limitReached = $derived(secrets.length >= maxSecrets);

	function formatDate(date: string | null) {
		if (!date) return m.never();
		return new Date(date).toLocaleString();
	}

	function expirationDate() {
		if (expiration === 'never') return null;
		const millisecondsPerDay = 24 * 60 * 60 * 1000;
		return new Date(Date.now() + Number(expiration) * millisecondsPerDay);
	}

	async function createSecret() {
		isCreating = true;
		try {
			const created = await oidcService.createClientSecret(client.id, expirationDate());
			// The value is kept in the store, so it survives switching between tabs but not leaving the page
			clientSecretStore.set(created.id, created.secret);
			secrets = [...secrets, created];
			expiration = 'never';
			toast.success(m.new_client_secret_created_successfully());
		} catch (e) {
			axiosErrorToast(e);
		} finally {
			isCreating = false;
		}
	}

	function deleteSecret(secret: OidcClientSecret) {
		openConfirmDialog({
			title: m.delete_client_secret(),
			message: m.are_you_sure_you_want_to_delete_this_client_secret(),
			confirm: {
				label: m.delete(),
				destructive: true,
				action: async () => {
					try {
						await oidcService.deleteClientSecret(client.id, secret.id);
						clientSecretStore.remove(secret.id);
						secrets = secrets.filter((s) => s.id !== secret.id);
						toast.success(m.client_secret_deleted_successfully());
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}
</script>

<Card.Root data-testid="client-secrets-card">
	<Card.Header>
		<Card.Title>{m.client_secrets()}</Card.Title>
		<Card.Description>{m.client_secrets_description()}</Card.Description>
	</Card.Header>
	<Card.Content>
		{#if client.isPublic}
			<p class="text-muted-foreground text-sm">{m.public_clients_cannot_have_secrets()}</p>
		{:else}
			{#if secrets.length > 0}
				<Table.Root>
					<Table.Header>
						<Table.Row>
							<Table.Head>{m.client_secret()}</Table.Head>
							<Table.Head>{m.created()}</Table.Head>
							<Table.Head>{m.expires_at()}</Table.Head>
							<Table.Head class="w-12"></Table.Head>
						</Table.Row>
					</Table.Header>
					<Table.Body>
						{#each secrets as secret (secret.id)}
							{@const revealed = $clientSecretStore[secret.id]}
							<Table.Row data-testid="client-secret-row">
								<Table.Cell>
									{#if revealed}
										<CopyToClipboard value={revealed}>
											<span class="font-mono text-sm break-all" data-testid="client-secret">
												{revealed}
											</span>
										</CopyToClipboard>
									{:else}
										<span
											class="text-muted-foreground font-mono text-sm"
											data-testid="client-secret">{secret.prefix}{'•'.repeat(8)}</span
										>
									{/if}
									{#if !secret.isActive}
										<Badge class="ml-2" variant="destructive">{m.expired()}</Badge>
									{/if}
								</Table.Cell>
								<Table.Cell class="text-muted-foreground text-sm"
									>{formatDate(secret.createdAt)}</Table.Cell
								>
								<Table.Cell class="text-muted-foreground text-sm"
									>{formatDate(secret.expiresAt)}</Table.Cell
								>
								<Table.Cell class="text-right">
									<DropdownMenu.Root>
										<DropdownMenu.Trigger
											class={buttonVariants({ variant: 'ghost', size: 'icon' })}
										>
											<LucideEllipsis class="size-4" />
											<span class="sr-only">{m.toggle_menu()}</span>
										</DropdownMenu.Trigger>
										<DropdownMenu.Content align="end">
											<DropdownMenu.Item class="text-red-500!" onclick={() => deleteSecret(secret)}>
												<LucideTrash2 class="mr-2 size-4" />
												{m.delete()}
											</DropdownMenu.Item>
										</DropdownMenu.Content>
									</DropdownMenu.Root>
								</Table.Cell>
							</Table.Row>
						{/each}
					</Table.Body>
				</Table.Root>
			{:else}
				<p class="text-muted-foreground text-sm">{m.no_client_secrets_yet()}</p>
			{/if}

			<div class="mt-5 flex flex-col justify-end gap-3 sm:flex-row sm:items-center">
				<Select.Root type="single" bind:value={expiration}>
					<Select.Trigger
						class="w-full sm:w-48"
						aria-label={m.expiration()}
						placeholder={m.expiration()}
					>
						{selectedExpirationLabel}
					</Select.Trigger>
					<Select.Content>
						{#each expirationOptions as option (option.value)}
							<Select.Item value={option.value}>{option.label}</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
				<Button onclick={createSecret} disabled={limitReached || isCreating}>
					<LucidePlus class="mr-2 size-4" />
					{m.add_client_secret()}
				</Button>
			</div>
			{#if limitReached}
				<p class="text-muted-foreground mt-2 text-end text-xs">
					{m.client_secrets_limit_reached()}
				</p>
			{/if}
		{/if}
	</Card.Content>
</Card.Root>
