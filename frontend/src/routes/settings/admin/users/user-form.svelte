<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import SwitchWithLabel from '$lib/components/form/switch-with-label.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Toggle } from '$lib/components/ui/toggle';
	import * as Tooltip from '$lib/components/ui/tooltip/index.js';
	import { m } from '$lib/paraglide/messages';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { User, UserCreate } from '$lib/types/user.type';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { emptyToUndefined, usernameSchema } from '$lib/utils/zod-util';
	import { LucideMailCheck, LucideMailWarning } from '@lucide/svelte';
	import { get } from 'svelte/store';
	import { z } from 'zod/v4';

	let {
		callback,
		existingUser,
		emailsVerifiedPerDefault = false
	}: {
		existingUser?: User;
		emailsVerifiedPerDefault?: boolean;
		callback: (user: UserCreate) => Promise<boolean>;
	} = $props();

	let isLoading = $state(false);
	let inputDisabled = $derived(!!existingUser?.ldapId && $appConfigStore.ldapEnabled);
	let hasManualDisplayNameEdit = $state(!!existingUser?.displayName);

	const user = {
		firstName: existingUser?.firstName || '',
		lastName: existingUser?.lastName || '',
		displayName: existingUser?.displayName || '',
		email: existingUser?.email || '',
		emailVerified: existingUser?.emailVerified ?? emailsVerifiedPerDefault,
		username: existingUser?.username || '',
		isAdmin: existingUser?.isAdmin || false,
		disabled: existingUser?.disabled || false
	};

	const formSchema = z.object({
		firstName: z.string().min(1).max(50),
		lastName: emptyToUndefined(z.string().max(50).optional()),
		displayName: z.string().min(1).max(100),
		username: usernameSchema,
		email: get(appConfigStore).requireUserEmail
			? z.email()
			: emptyToUndefined(z.email().optional()),
		emailVerified: z.boolean(),
		isAdmin: z.boolean(),
		disabled: z.boolean()
	});
	type FormSchema = typeof formSchema;

	const { inputs, ...form } = createForm<FormSchema>(formSchema, user);
	async function onSubmit() {
		const data = form.validate();
		if (!data) return;
		isLoading = true;
		const success = await callback(data);
		// Reset form if user was successfully created
		if (success && !existingUser) form.reset();
		isLoading = false;
	}
	function onNameInput() {
		if (!hasManualDisplayNameEdit) {
			$inputs.displayName.value = `${$inputs.firstName.value}${
				$inputs.lastName?.value ? ' ' + $inputs.lastName.value : ''
			}`;
		}
	}
</script>

<form onsubmit={preventDefault(onSubmit)}>
	<fieldset disabled={inputDisabled}>
		<div class="grid grid-cols-1 items-start gap-5 md:grid-cols-2">
			<FormInput label={m.first_name()} oninput={onNameInput} bind:input={$inputs.firstName} />
			<FormInput label={m.last_name()} oninput={onNameInput} bind:input={$inputs.lastName} />
			<FormInput
				label={m.display_name()}
				oninput={() => (hasManualDisplayNameEdit = true)}
				bind:input={$inputs.displayName}
			/>
			<FormInput label={m.username()} bind:input={$inputs.username} />
			<div class="flex items-end">
				<FormInput
					inputClass="rounded-r-none border-r-0"
					label={m.email()}
					bind:input={$inputs.email}
				/>
				<Tooltip.Provider>
					{@const label = $inputs.emailVerified.value
						? m.mark_as_unverified()
						: m.mark_as_verified()}
					<Tooltip.Root>
						<Tooltip.Trigger>
							<Toggle
								bind:pressed={$inputs.emailVerified.value}
								aria-label={label}
								class="h-9 border-input bg-yellow-100 dark:bg-yellow-950 data-[state=on]:bg-green-100 dark:data-[state=on]:bg-green-950 rounded-l-none border px-2 py-1 shadow-xs flex items-center hover:data-[state=on]:bg-accent"
							>
								{#if $inputs.emailVerified.value}
									<LucideMailCheck class="text-green-500 dark:text-green-600 size-5" />
								{:else}
									<LucideMailWarning class="text-yellow-500 dark:text-yellow-600 size-5" />
								{/if}
							</Toggle>
						</Tooltip.Trigger>
						<Tooltip.Content>{label}</Tooltip.Content>
					</Tooltip.Root>
				</Tooltip.Provider>
			</div>
		</div>
		<div class="mt-5 grid grid-cols-1 items-start gap-5 md:grid-cols-2">
			<SwitchWithLabel
				id="admin-privileges"
				label={m.admin_privileges()}
				description={m.admins_have_full_access_to_the_admin_panel()}
				bind:checked={$inputs.isAdmin.value}
			/>
			<SwitchWithLabel
				id="user-disabled"
				label={m.user_disabled()}
				description={m.disabled_users_cannot_log_in_or_use_services()}
				bind:checked={$inputs.disabled.value}
			/>
		</div>
		<div class="mt-5 flex justify-end">
			<Button {isLoading} type="submit">{m.save()}</Button>
		</div>
	</fieldset>
</form>
