<script lang="ts">
	import { page } from '$app/state';
	import CopyToClipboard from '$lib/components/copy-to-clipboard.svelte';
	import Qrcode from '$lib/components/qrcode/qrcode.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Input } from '$lib/components/ui/input';
	import Label from '$lib/components/ui/label/label.svelte';
	import * as Select from '$lib/components/ui/select/index.js';
	import { Separator } from '$lib/components/ui/separator';
	import { m } from '$lib/paraglide/messages';
	import UserService from '$lib/services/user-service';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { mode } from 'mode-watcher';

	let {
		open = $bindable(),
		onTokenCreated
	}: {
		open: boolean;
		onTokenCreated?: () => Promise<void>;
	} = $props();

	const userService = new UserService();

	let signupToken: string | null = $state(null);
	let signupLink: string | null = $state(null);
	let selectedExpiration: keyof typeof availableExpirations = $state(m.one_day());
	let usageLimit: number = $state(1);

	let availableExpirations = {
		[m.one_hour()]: 60 * 60,
		[m.twelve_hours()]: 60 * 60 * 12,
		[m.one_day()]: 60 * 60 * 24,
		[m.one_week()]: 60 * 60 * 24 * 7,
		[m.one_month()]: 60 * 60 * 24 * 30
	};

	async function createSignupToken() {
		try {
			const expiration = new Date(Date.now() + availableExpirations[selectedExpiration] * 1000);
			signupToken = await userService.createSignupToken(expiration, usageLimit);
			signupLink = `${page.url.origin}/st/${signupToken}`;

			if (onTokenCreated) {
				await onTokenCreated();
			}
		} catch (e) {
			axiosErrorToast(e);
		}
	}

	function onOpenChange(isOpen: boolean) {
		open = isOpen;
		if (!isOpen) {
			signupToken = null;
			signupLink = null;
			selectedExpiration = m.one_day();
			usageLimit = 1;
		}
	}
</script>

<Dialog.Root {open} {onOpenChange}>
	<Dialog.Content class="max-w-md">
		<Dialog.Header>
			<Dialog.Title>{m.signup_token()}</Dialog.Title>
			<Dialog.Description
				>{m.create_a_signup_token_to_allow_new_user_registration()}</Dialog.Description
			>
		</Dialog.Header>

		{#if signupToken === null}
			<div class="space-y-4">
				<div>
					<Label for="expiration">{m.expiration()}</Label>
					<Select.Root
						type="single"
						value={Object.keys(availableExpirations)[0]}
						onValueChange={(v) => (selectedExpiration = v! as keyof typeof availableExpirations)}
					>
						<Select.Trigger id="expiration" class="h-9 w-full">
							{selectedExpiration}
						</Select.Trigger>
						<Select.Content>
							{#each Object.keys(availableExpirations) as key}
								<Select.Item value={key}>{key}</Select.Item>
							{/each}
						</Select.Content>
					</Select.Root>
				</div>

				<div>
					<Label for="usage-limit">{m.usage_limit()}</Label>
					<Input
						id="usage-limit"
						type="number"
						min="1"
						max="100"
						bind:value={usageLimit}
						class="h-9"
					/>
					<p class="text-muted-foreground mt-1 text-xs">
						{m.number_of_times_token_can_be_used()}
					</p>
				</div>
			</div>

			<Dialog.Footer class="mt-4">
				<Button
					onclick={() => createSignupToken()}
					disabled={!selectedExpiration || usageLimit < 1}
				>
					{m.create()}
				</Button>
			</Dialog.Footer>
		{:else}
			<div class="flex flex-col items-center gap-2">
				<CopyToClipboard value={signupToken!}>
					<p class="px-2 text-center text-2xl font-semibold break-all">{signupToken}</p>
				</CopyToClipboard>

				<div class="text-muted-foreground my-2 flex items-center justify-center gap-3">
					<Separator />
					<p class="text-xs text-nowrap">{m.or_visit()}</p>
					<Separator />
				</div>

				<Qrcode
					class="mb-2"
					value={signupLink}
					size={180}
					color={mode.current === 'dark' ? '#FFFFFF' : '#000000'}
					backgroundColor={mode.current === 'dark' ? '#000000' : '#FFFFFF'}
				/>
				<CopyToClipboard value={signupLink!}>
					<p data-testId="signup-token-link" class="px-2 text-center text-sm break-all">
						{signupLink!}
					</p>
				</CopyToClipboard>

				<div class="text-muted-foreground mt-2 text-center text-sm">
					<p>{m.usage_limit()}: {usageLimit}</p>
					<p>{m.expiration()}: {selectedExpiration}</p>
				</div>
			</div>
		{/if}
	</Dialog.Content>
</Dialog.Root>
