<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import WebAuthnService from '$lib/services/webauthn-service';
	import type { UserCreate } from '$lib/types/user.type';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { getWebauthnErrorMessage } from '$lib/utils/error-util';
	import { startRegistration } from '@simplewebauthn/browser';
	import { z } from 'zod/v4';
	import { goto } from '$app/navigation';
	import { tryCatch } from '$lib/utils/try-catch-util';

	let {
		callback,
		isLoading
	}: {
		callback: (user: UserCreate) => Promise<boolean>;
		isLoading: boolean;
	} = $props();

	const webauthnService = new WebAuthnService();

	const initialData = {
		firstName: '',
		lastName: '',
		email: '',
		username: '',
		isAdmin: false,
		disabled: false
	};

	const formSchema = z.object({
		firstName: z.string().min(1, m.first_name_required()).max(50),
		lastName: z.string().max(50),
		username: z
			.string()
			.min(2, m.username_must_be_at_least_2_characters())
			.max(30)
			.regex(/^[a-z0-9_@.-]+$/, m.username_can_only_contain()),
		email: z.email(m.please_enter_a_valid_email()),
		isAdmin: z.boolean(),
		disabled: z.boolean()
	});
	type FormSchema = typeof formSchema;

	const { inputs, ...form } = createForm<FormSchema>(formSchema, initialData);

	let step = $state<'form' | 'passkey'>('form');
	let signupError = $state<string | undefined>();
	let userData: UserCreate | null = $state(null);

	async function onSubmit() {
		const data = form.validate();
		if (!data) return;

		isLoading = true;
		const result = await tryCatch(callback(data));
		if (result.data) {
			userData = data;
			step = 'passkey';
			isLoading = false;
		}
	}

	async function createPasskeyAndContinue() {
		if (!userData) return;

		isLoading = true;
		signupError = undefined;

		const optsResult = await tryCatch(webauthnService.getRegistrationOptions());
		if (optsResult.error) {
			signupError = getWebauthnErrorMessage(optsResult.error);
			isLoading = false;
			return;
		}

		const attRespResult = await tryCatch(startRegistration({ optionsJSON: optsResult.data }));
		if (attRespResult.error) {
			signupError = getWebauthnErrorMessage(attRespResult.error);
			isLoading = false;
			return;
		}

		const finishResult = await tryCatch(webauthnService.finishRegistration(attRespResult.data));
		if (finishResult.error) {
			signupError = getWebauthnErrorMessage(finishResult.error);
			isLoading = false;
			return;
		}

		goto('/settings/account');
		isLoading = false;
	}

	function skipPasskeySetup() {
		goto('/settings/account');
	}
</script>

{#if step === 'form'}
	<form onsubmit={preventDefault(onSubmit)} class="w-full max-w-[450px]">
		<div class="mt-7 space-y-4">
			<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
				<FormInput label={m.first_name()} bind:input={$inputs.firstName} />
				<FormInput label={m.last_name()} bind:input={$inputs.lastName} />
			</div>

			<FormInput label={m.username()} bind:input={$inputs.username} />
			<FormInput label={m.email()} bind:input={$inputs.email} type="email" />
		</div>

		<div class="mt-8 flex justify-between gap-2">
			<Button variant="secondary" class="flex-1" href="/login">{m.go_to_login()}</Button>
			<Button class="flex-1" type="submit" disabled={isLoading}>
				{m.continue()}
			</Button>
		</div>
	</form>
{:else if step === 'passkey'}
	<div class="w-full max-w-[450px] text-center">
		<div class="mt-7 space-y-4">
			<div class="bg-muted rounded-lg p-6">
				<h3 class="mb-2 text-lg font-semibold">{m.setup_your_passkey()}</h3>
				<p class="text-muted-foreground mb-4 text-sm">
					{m.create_a_passkey_to_securely_access_your_account()}
				</p>

				{#if signupError}
					<div class="bg-destructive/10 border-destructive/20 mb-4 rounded-md border p-3">
						<p class="text-destructive text-sm">{signupError}</p>
					</div>
				{/if}

				<div class="flex flex-col gap-2">
					<Button onclick={createPasskeyAndContinue} {isLoading} class="w-full">
						{m.add_passkey()}
					</Button>
					<Button variant="ghost" onclick={skipPasskeySetup} disabled={isLoading} class="w-full">
						{m.skip_for_now()}
					</Button>
				</div>
			</div>
		</div>
	</div>
{/if}
