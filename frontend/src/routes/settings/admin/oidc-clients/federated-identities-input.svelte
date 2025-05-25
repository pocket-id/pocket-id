<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import Label from '$lib/components/ui/label/label.svelte';
	import { m } from '$lib/paraglide/messages';
	import type { OidcClientFederatedIdentity } from '$lib/types/oidc.type';
	import { LucideMinus, LucidePlus } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		federatedIdentities = $bindable([]),
		error = $bindable(null),
		...restProps
	}: HTMLAttributes<HTMLDivElement> & {
		federatedIdentities: OidcClientFederatedIdentity[];
		error?: string | null;
		children?: Snippet;
	} = $props();

	function addFederatedIdentity() {
		federatedIdentities = [
			...federatedIdentities,
			{
				issuer: '',
				subject: '',
				audience: '',
				jwks: ''
			}
		];
	}

	function removeFederatedIdentity(index: number) {
		federatedIdentities = federatedIdentities.filter((_, i) => i !== index);
	}

	function updateFederatedIdentity(
		index: number,
		field: keyof OidcClientFederatedIdentity,
		value: string
	) {
		federatedIdentities[index] = {
			...federatedIdentities[index],
			[field]: value
		};
	}
</script>

<div {...restProps}>
	<FormInput label={m.federated_identities()} description={m.federated_identities_description()}>
		<div class="space-y-4">
			{#each federatedIdentities as identity, i}
				<div class="space-y-3 rounded-lg border p-4">
					<div class="flex items-center justify-between">
						<Label class="text-sm font-medium">Identity {i + 1}</Label>
						{#if federatedIdentities.length > 0}
							<Button
								variant="outline"
								size="sm"
								onclick={() => removeFederatedIdentity(i)}
								aria-label="Remove federated identity"
							>
								<LucideMinus class="size-4" />
							</Button>
						{/if}
					</div>

					<div class="grid grid-cols-1 gap-3 md:grid-cols-2">
						<div>
							<Label for="issuer-{i}" class="text-xs">Issuer (Required)</Label>
							<Input
								id="issuer-{i}"
								placeholder="https://example.com/"
								value={identity.issuer}
								oninput={(e) => updateFederatedIdentity(i, 'issuer', e.currentTarget.value)}
								required
							/>
						</div>

						<div>
							<Label for="subject-{i}" class="text-xs">Subject (Optional)</Label>
							<Input
								id="subject-{i}"
								placeholder="Defaults to client ID"
								value={identity.subject || ''}
								oninput={(e) => updateFederatedIdentity(i, 'subject', e.currentTarget.value)}
							/>
						</div>

						<div>
							<Label for="audience-{i}" class="text-xs">Audience (Optional)</Label>
							<Input
								id="audience-{i}"
								placeholder="Defaults to Pocket ID URL"
								value={identity.audience || ''}
								oninput={(e) => updateFederatedIdentity(i, 'audience', e.currentTarget.value)}
							/>
						</div>

						<div>
							<Label for="jwks-{i}" class="text-xs">JWKS URL (Optional)</Label>
							<Input
								id="jwks-{i}"
								placeholder="Defaults to issuer/.well-known/jwks.json"
								value={identity.jwks || ''}
								oninput={(e) => updateFederatedIdentity(i, 'jwks', e.currentTarget.value)}
							/>
						</div>
					</div>
				</div>
			{/each}
		</div>
	</FormInput>

	{#if error}
		<p class="text-destructive mt-1 text-xs">{error}</p>
	{/if}

	<Button class="mt-3" variant="secondary" size="sm" onclick={addFederatedIdentity} type="button">
		<LucidePlus class="mr-1 size-4" />
		{federatedIdentities.length === 0
			? m.add_federated_identity()
			: m.add_another_federated_identity()}
	</Button>
</div>
