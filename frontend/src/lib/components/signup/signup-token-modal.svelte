<script lang="ts">
	import { page } from '$app/state';
	import CopyToClipboard from '$lib/components/copy-to-clipboard.svelte';
	import FormInput from '$lib/components/form/form-input.svelte';
	import UserGroupInput from '$lib/components/form/user-group-input.svelte';
	import Qrcode from '$lib/components/qrcode/qrcode.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Input } from '$lib/components/ui/input';
	import * as Select from '$lib/components/ui/select/index.js';
	import { m } from '$lib/paraglide/messages';
	import AppConfigService from '$lib/services/app-config-service';
	import UserService from '$lib/services/user-service';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { mode } from 'mode-watcher';
	import { onMount } from 'svelte';
	import { z } from 'zod/v4';

	let {
		open = $bindable()
	}: {
		open: boolean;
	} = $props();

	const userService = new UserService();
	const appConfigService = new AppConfigService();

	const DEFAULT_TTL_SECONDS = 60 * 60 * 24;
	const availableExpirations = [
		{ label: m.one_hour(), value: 60 * 60 },
		{ label: m.twelve_hours(), value: 60 * 60 * 12 },
		{ label: m.one_day(), value: DEFAULT_TTL_SECONDS },
		{ label: m.one_week(), value: DEFAULT_TTL_SECONDS * 7 },
		{ label: m.one_month(), value: DEFAULT_TTL_SECONDS * 30 }
	] as const;

	const defaultExpiration =
		availableExpirations.find((exp) => exp.value === DEFAULT_TTL_SECONDS)?.value ??
		availableExpirations[0].value;

	type SignupTokenForm = {
		ttl: number;
		usageLimit: number;
		userGroupIds: string[];
	};

	const initialFormValues: SignupTokenForm = {
		ttl: defaultExpiration,
		usageLimit: 1,
		userGroupIds: []
	};

	const formSchema = z.object({
		ttl: z.number(),
		usageLimit: z.number().min(1).max(100),
		userGroupIds: z.array(z.string()).default([])
	});

	const { inputs, ...form } = createForm<typeof formSchema>(formSchema, initialFormValues);

	let signupToken: string | null = $state(null);
	let signupLink: string | null = $state(null);
	let createdSignupData: SignupTokenForm | null = $state(null);
	let isLoading = $state(false);

	let defaultUserGroupIds: string[] = [];

	function getExpirationLabel(ttl: number) {
		return availableExpirations.find((exp) => exp.value === ttl)?.label ?? '';
	}

	function resetForm() {
		form.reset();
		form.setValue('userGroupIds', defaultUserGroupIds);
	}

	async function createSignupToken() {
		const data = form.validate();
		if (!data) return;

		isLoading = true;
		try {
			signupToken = await userService.createSignupToken(
				data.ttl,
				data.usageLimit,
				data.userGroupIds
			);
			signupLink = `${page.url.origin}/st/${signupToken}`;
			createdSignupData = data;
		} catch (e) {
			axiosErrorToast(e);
		} finally {
			isLoading = false;
		}
	}

	function onOpenChange(isOpen: boolean) {
		open = isOpen;
		if (!isOpen) {
			signupToken = null;
			signupLink = null;
			createdSignupData = null;
			resetForm();
		}
	}

	onMount(() => {
		appConfigService
			.list(true)
			.then((response) => {
				const responseGroupIds = response.signupDefaultUserGroupIDs || [];
				defaultUserGroupIds = responseGroupIds;
				initialFormValues.userGroupIds = responseGroupIds;
				form.setValue('userGroupIds', responseGroupIds);
			})
			.catch(axiosErrorToast);
	});
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
			<form class="space-y-4" onsubmit={preventDefault(createSignupToken)}>
				<FormInput labelFor="expiration" label={m.expiration()} input={$inputs.ttl}>
					<Select.Root
						type="single"
						value={$inputs.ttl.value.toString()}
						onValueChange={(v) => v && form.setValue('ttl', Number(v))}
					>
						<Select.Trigger id="expiration" class="w-full">
							{getExpirationLabel($inputs.ttl.value)}
						</Select.Trigger>
						<Select.Content>
							{#each availableExpirations as expiration}
								<Select.Item value={expiration.value.toString()}>
									{expiration.label}
								</Select.Item>
							{/each}
						</Select.Content>
					</Select.Root>
					{#if $inputs.ttl.error}
						<p class="text-destructive mt-1 text-xs">{$inputs.ttl.error}</p>
					{/if}
				</FormInput>
				<FormInput
					labelFor="usage-limit"
					label={m.usage_limit()}
					description={m.number_of_times_token_can_be_used()}
					input={$inputs.usageLimit}
				>
					<Input
						id="usage-limit"
						type="number"
						bind:value={$inputs.usageLimit.value}
						aria-invalid={$inputs.usageLimit.error ? 'true' : undefined}
						class="h-9"
					/>
				</FormInput>
				<FormInput
					labelFor="default-groups"
					label={m.user_groups()}
					description={m.signup_token_user_groups_description()}
					input={$inputs.userGroupIds}
				>
					<UserGroupInput bind:selectedGroupIds={$inputs.userGroupIds.value} />
				</FormInput>

				<Dialog.Footer class="mt-4">
					<Button type="submit" {isLoading}>
						{m.create()}
					</Button>
				</Dialog.Footer>
			</form>
		{:else}
			<div class="flex flex-col items-center gap-2">
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
					<p>{m.usage_limit()}: {createdSignupData?.usageLimit}</p>
					<p>{m.expiration()}: {getExpirationLabel(createdSignupData?.ttl ?? 0)}</p>
				</div>
			</div>
		{/if}
	</Dialog.Content>
</Dialog.Root>
