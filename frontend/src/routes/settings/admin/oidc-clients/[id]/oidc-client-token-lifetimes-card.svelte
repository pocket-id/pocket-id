<script lang="ts">
	import DurationInput from '$lib/components/form/duration-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { m } from '$lib/paraglide/messages';
	import type { OidcClient, OidcClientTokenLifetimes } from '$lib/types/oidc.type';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { z } from 'zod/v4';

	let {
		client,
		callback
	}: {
		client: OidcClient;
		callback: (lifetimes: OidcClientTokenLifetimes) => Promise<boolean>;
	} = $props();

	let isLoading = $state(false);

	const durationSchema = z
		.number()
		.min(60, { message: m.token_lifetime_minimum() })
		.max(365 * 24 * 60 * 60, { message: m.token_lifetime_maximum() })
		.refine((seconds) => Number.isInteger(seconds) && seconds % 60 === 0, {
			message: m.token_lifetime_whole_minutes()
		});
	const formSchema = z.object({
		accessTokenDurationSeconds: durationSchema,
		refreshTokenDurationSeconds: durationSchema
	});
	const { inputs, ...form } = createForm(formSchema, {
		accessTokenDurationSeconds: client.accessTokenDurationSeconds,
		refreshTokenDurationSeconds: client.refreshTokenDurationSeconds
	});

	async function onSubmit() {
		const data = form.validate();
		if (!data) return;

		isLoading = true;
		await callback(data).finally(() => (isLoading = false));
	}
</script>

<form novalidate onsubmit={preventDefault(onSubmit)}>
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
					bind:input={$inputs.accessTokenDurationSeconds}
				/>
				<DurationInput
					id="refresh-token-lifetime"
					label={m.refresh_token_inactivity_timeout()}
					description={m.refresh_token_inactivity_timeout_description()}
					bind:input={$inputs.refreshTokenDurationSeconds}
				/>
			</div>
		</Card.Content>
		<Card.Footer class="justify-end">
			<Button type="submit" disabled={isLoading}>{m.save()}</Button>
		</Card.Footer>
	</Card.Root>
</form>
