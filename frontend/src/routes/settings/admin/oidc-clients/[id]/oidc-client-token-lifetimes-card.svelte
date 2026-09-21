<script lang="ts">
	import DurationInput from '$lib/components/form/duration-input.svelte';
	import * as Card from '$lib/components/ui/card';
	import { m } from '$lib/paraglide/messages';
	import type { OidcClient, OidcClientTokenLifetimes } from '$lib/types/oidc.type';
	import { createForm } from '$lib/utils/form-util';
	import { trackFormChanges } from '$lib/utils/unsaved-changes-util.svelte';
	import { z } from 'zod/v4';

	let {
		client,
		callback
	}: {
		client: OidcClient;
		callback: (lifetimes: OidcClientTokenLifetimes) => Promise<void>;
	} = $props();

	const durationSchema = z
		.number()
		.min(1, { message: m.token_lifetime_minimum() })
		.max(365 * 24 * 60, { message: m.token_lifetime_maximum() })
		.refine((minutes) => Number.isInteger(minutes), {
			message: m.token_lifetime_whole_minutes()
		});
	const formSchema = z.object({
		accessTokenDurationMinutes: durationSchema,
		refreshTokenDurationMinutes: durationSchema
	});
	const formStore = createForm(formSchema, {
		accessTokenDurationMinutes: client.accessTokenDurationMinutes,
		refreshTokenDurationMinutes: client.refreshTokenDurationMinutes
	});
	const { inputs } = formStore;

	trackFormChanges(() => formStore, callback);
</script>

<Card.Root data-testid="token-lifetimes-card">
	<Card.Header>
		<Card.Title>{m.token_lifetimes()}</Card.Title>
		<Card.Description>{m.token_lifetimes_description()}</Card.Description>
	</Card.Header>
	<Card.Content>
		<div class="md:grid md:grid-cols-2 gap-10 space-y-5 md:space-y-0">
			<DurationInput
				id="access-token-lifetime"
				label={m.access_token_lifetime()}
				description={m.access_token_lifetime_description()}
				bind:input={$inputs.accessTokenDurationMinutes}
			/>
			<DurationInput
				id="refresh-token-lifetime"
				label={m.refresh_token_inactivity_timeout()}
				description={m.refresh_token_inactivity_timeout_description()}
				bind:input={$inputs.refreshTokenDurationMinutes}
			/>
		</div>
	</Card.Content>
</Card.Root>
