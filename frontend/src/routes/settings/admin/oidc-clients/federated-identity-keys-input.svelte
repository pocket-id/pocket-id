<script lang="ts">
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Field from '#lib/components/ui/field/index.ts';
	import { Textarea } from '#lib/components/ui/textarea/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { describeJwk, getJwkKeyId, parseJwkInput, type Jwk } from '#lib/utils/jwk-util.ts';
	import { LucidePlus, LucideX } from '@lucide/svelte';

	let {
		publicKeys,
		onChange,
		id,
		disabled = false
	}: {
		publicKeys: Jwk[];
		onChange: (publicKeys: Jwk[]) => void;
		id: string;
		disabled?: boolean;
	} = $props();

	let pastedKey = $state('');
	let error = $state<string | null>(null);

	function addKeys() {
		const result = parseJwkInput(pastedKey);
		if (!result.ok) {
			error = result.error;
			return;
		}

		// A JWKS may repeat a key that was already added, and duplicate key IDs would make it ambiguous which key verifies an assertion
		const existingKeyIds = new Set(publicKeys.map(getJwkKeyId));
		const duplicate = result.keys.find((key) => existingKeyIds.has(getJwkKeyId(key)));
		if (duplicate) {
			error = m.public_key_already_added({ keyId: getJwkKeyId(duplicate) });
			return;
		}

		onChange([...publicKeys, ...result.keys]);
		pastedKey = '';
		error = null;
	}

	function removeKey(index: number) {
		onChange(publicKeys.filter((_, i) => i !== index));
	}
</script>

<div class="flex flex-col gap-3">
	{#if publicKeys.length > 0}
		<ul class="flex flex-col rounded-md border" aria-label={m.public_keys()}>
			{#each publicKeys as key, i (getJwkKeyId(key))}
				<li
					class="flex min-h-11 items-center justify-between gap-3 border-b py-1.5 ps-4 pe-2 last:border-b-0"
					data-testid="federated-identity-public-key"
				>
					<div class="flex min-w-0 flex-col">
						<span class="text-mono-s truncate font-mono">{getJwkKeyId(key)}</span>
						<span class="text-caption text-fg-muted">{describeJwk(key)}</span>
					</div>
					{#if !disabled}
						<Button
							variant="ghost"
							size="icon-xs"
							onclick={() => removeKey(i)}
							aria-label={m.remove_public_key({ keyId: getJwkKeyId(key) })}
						>
							<LucideX />
						</Button>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}

	{#if !disabled}
		<Field.Field>
			<Field.Label for={id}>{m.public_key()}</Field.Label>
			<Textarea
				{id}
				class="font-mono"
				rows={4}
				placeholder={'{"kty": "RSA", "kid": "...", "n": "...", "e": "AQAB"}'}
				bind:value={pastedKey}
				aria-invalid={!!error}
				oninput={() => (error = null)}
			/>
			{#if error}
				<Field.Error>{error}</Field.Error>
			{:else}
				<Field.Description>{m.paste_public_key_description()}</Field.Description>
			{/if}
		</Field.Field>

		<div>
			<Button
				variant="secondary"
				size="sm"
				type="button"
				onclick={addKeys}
				disabled={!pastedKey.trim()}
			>
				<LucidePlus />
				{m.add_public_key()}
			</Button>
		</div>
	{/if}
</div>
