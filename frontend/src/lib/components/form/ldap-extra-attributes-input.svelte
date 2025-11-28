<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import type { LdapExtraAttribute } from '$lib/types/application-configuration';
	import SwitchWithLabel from '$lib/components/form/switch-with-label.svelte';
	import { LucideMinus, LucidePlus } from '@lucide/svelte';
	import { onMount, type Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';
	import { m } from '$lib/paraglide/messages';

	let {
		ldapExtraAttributes = $bindable(),
		error = $bindable(null)
	}: HTMLAttributes<HTMLDivElement> & {
		ldapExtraAttributes: LdapExtraAttribute[];
		error?: string | null;
		children?: Snippet;
	} = $props();

	const limit = 20;
</script>

<div>
	<FormInput>
		<div class="flex flex-col gap-y-2">
			{#each ldapExtraAttributes as _, i}
				<div class="flex gap-x-2">
					<Input
						placeholder={m.ldap_attribute_key()}
						bind:value={ldapExtraAttributes[i].key}
					/>
					<Input placeholder={m.ldap_attribute_name()} bind:value={ldapExtraAttributes[i].value} />
          <SwitchWithLabel
            id="multi"
            label={m.ldap_mutli_valued_attribute()}
            bind:checked={ldapExtraAttributes[i].multi}
          />
					<Button
						variant="outline"
						size="sm"
						aria-label={m.remove_custom_claim()}
						onclick={() => (ldapExtraAttributes = ldapExtraAttributes.filter((_, index) => index !== i))}
          >
            <LucideMinus class="size-4" />
					</Button>
				</div>
			{/each}
		</div>
	</FormInput>
	{#if error}
		<p class="text-destructive mt-1 text-xs">{error}</p>
	{/if}
	{#if ldapExtraAttributes.length < limit}
		<Button
			class="mt-2"
			variant="secondary"
			size="sm"
			onclick={() => (ldapExtraAttributes = [...ldapExtraAttributes, { key: '', value: '', multi: false }])}
		>
			<LucidePlus class="mr-1 size-4" />
			{ldapExtraAttributes.length === 0 ? m.add_ldap_attribute() : m.add_another()}
		</Button>
	{/if}
</div>
