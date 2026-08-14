<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Item from '$lib/components/ui/item/index.js';
	import * as Tooltip from '$lib/components/ui/tooltip/index.js';
	import { m } from '$lib/paraglide/messages';
	import type { RuntimeCredential } from '$lib/types/runtime-credential.type';
	import { Bot, LucidePencil, LucideTrash } from '@lucide/svelte';

	let {
		credentials,
		onRevoke,
		onRename
	}: {
		credentials: RuntimeCredential[];
		onRevoke: (credential: RuntimeCredential) => void;
		onRename?: (credential: RuntimeCredential) => void;
	} = $props();
</script>

<Item.Group class="mt-3">
	{#each [...credentials].sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()) as credential (credential.id)}
		<Item.Root variant="transparent" class="hover:bg-muted px-0 py-3 transition-colors sm:px-4">
			<Item.Media class="bg-primary/10 text-primary rounded-full p-3">
				<Bot class="size-5" />
			</Item.Media>
			<Item.Content class="gap-0.5">
				<Item.Title>{credential.name}</Item.Title>
				<Item.Description class="space-y-0.5">
					<span class="block font-mono text-xs">{m.credential_id_value({ id: credential.id })}</span
					>
					<span class="block">
						{m.runtime_credential_algorithm_created({
							algorithm: credential.algorithm,
							date: new Date(credential.createdAt).toLocaleString()
						})}
					</span>
					<span class="block">
						{credential.revokedAt
							? m.runtime_credential_revoked_on({
									date: new Date(credential.revokedAt).toLocaleString()
								})
							: credential.lastUsedAt
								? m.runtime_credential_last_used({
										date: new Date(credential.lastUsedAt).toLocaleString()
									})
								: m.runtime_credential_never_used()}
					</span>
					<span class="block">
						{credential.expiresAt
							? m.runtime_credential_expires_on({
									date: new Date(credential.expiresAt).toLocaleString()
								})
							: m.runtime_credential_no_expiration()}
					</span>
				</Item.Description>
			</Item.Content>
			{#if !credential.revokedAt}
				<Item.Actions>
					{#if onRename}
						<Tooltip.Provider>
							<Tooltip.Root>
								<Tooltip.Trigger>
									<Button
										onclick={() => onRename?.(credential)}
										size="icon"
										variant="ghost"
										class="size-8"
										aria-label={m.rename()}
									>
										<LucidePencil class="size-4" />
									</Button>
								</Tooltip.Trigger>
								<Tooltip.Content>{m.rename()}</Tooltip.Content>
							</Tooltip.Root>
						</Tooltip.Provider>
					{/if}
					<Tooltip.Provider>
						<Tooltip.Root>
							<Tooltip.Trigger>
								<Button
									onclick={() => onRevoke(credential)}
									size="icon"
									variant="ghost"
									class="hover:bg-destructive/10 hover:text-destructive size-8"
									aria-label={m.revoke()}
								>
									<LucideTrash class="size-4" />
								</Button>
							</Tooltip.Trigger>
							<Tooltip.Content>{m.revoke()}</Tooltip.Content>
						</Tooltip.Root>
					</Tooltip.Provider>
				</Item.Actions>
			{/if}
		</Item.Root>
	{/each}
</Item.Group>
