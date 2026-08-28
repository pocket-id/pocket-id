<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Field from '$lib/components/ui/field';
	import { Textarea } from '$lib/components/ui/textarea';
	import { m } from '$lib/paraglide/messages';
	import { describeJwk, getJwkKeyId, parseJwkInput, type Jwk } from '$lib/utils/jwk-util';
	import { LucideTrash2 } from '@lucide/svelte';

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

	// The list is owned by this component: the identity it belongs to lives in a store, whose
	// updates don't propagate back down to this input
	let keys = $state<Jwk[]>(publicKeys);
	let pastedKey = $state('');
	let error = $state<string | null>(null);

	function addKeys() {
		const result = parseJwkInput(pastedKey);
		if (!result.ok) {
			error = result.error;
			return;
		}

		// A JWKS may repeat a key that was already added, and duplicate key IDs would make the key
		// to verify an assertion with ambiguous
		const existingKeyIds = new Set(keys.map(getJwkKeyId));
		const duplicate = result.keys.find((key) => existingKeyIds.has(getJwkKeyId(key)));
		if (duplicate) {
			error = m.public_key_already_added({ keyId: getJwkKeyId(duplicate) });
			return;
		}

		keys = [...keys, ...result.keys];
		onChange(keys);
		pastedKey = '';
		error = null;
	}

	function removeKey(index: number) {
		keys = keys.filter((_, i) => i !== index);
		onChange(keys);
	}
</script>

<div class="flex flex-col gap-3">
	{#if keys.length > 0}
		<ul class="flex flex-col gap-2" data-testid="federated-identity-public-keys">
			{#each keys as key, i (getJwkKeyId(key))}
				<li
					class="bg-muted/40 flex items-center justify-between gap-3 rounded-2xl px-4 py-2"
					data-testid="federated-identity-public-key"
				>
					<div class="flex min-w-0 flex-col">
						<span class="truncate font-mono text-sm">{getJwkKeyId(key)}</span>
						<span class="text-muted-foreground text-[0.8rem]">{describeJwk(key)}</span>
					</div>
					<Button
						variant="ghost"
						size="sm"
						onclick={() => removeKey(i)}
						aria-label={m.remove_public_key({ keyId: getJwkKeyId(key) })}
						{disabled}
					>
						<LucideTrash2 />
					</Button>
				</li>
			{/each}
		</ul>
	{/if}

	<Field.Field>
		<Field.Label for={id}>{m.public_key()}</Field.Label>
		<Textarea
			{id}
			class="font-mono text-xs"
			rows={4}
			placeholder={'{"kty": "RSA", "kid": "...", "n": "...", "e": "AQAB"}'}
			bind:value={pastedKey}
			aria-invalid={!!error}
			oninput={() => (error = null)}
			{disabled}
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
			disabled={disabled || !pastedKey.trim()}
		>
			{m.add_public_key()}
		</Button>
	</div>
</div>
