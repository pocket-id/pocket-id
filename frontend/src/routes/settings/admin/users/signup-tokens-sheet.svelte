<script lang="ts">
	import { page } from '$app/state';
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
	import CopyField from '#lib/components/copy-field.svelte';
	import FormInput from '#lib/components/form/form-input.svelte';
	import UserGroupInput from '#lib/components/form/user-group-input.svelte';
	import Qrcode from '#lib/components/qrcode/qrcode.svelte';
	import { Badge } from '@pocket-id/ui/badge';
	import { Button } from '@pocket-id/ui/button';
	import { Input } from '@pocket-id/ui/input';
	import * as Select from '@pocket-id/ui/select';
	import * as Sheet from '@pocket-id/ui/sheet';
	import { Skeleton } from '@pocket-id/ui/skeleton';
	import { m } from '#lib/paraglide/messages.js';
	import { getLocale } from '#lib/paraglide/runtime.js';
	import AppConfigService from '#lib/services/app-config-service.ts';
	import UserService from '#lib/services/user-service.ts';
	import type { SignupToken } from '#lib/types/signup-token.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { preventDefault } from '#lib/utils/event-util.ts';
	import { createForm } from '#lib/utils/form-util.ts';
	import { cn } from '@pocket-id/ui/utils';
	import { emptyToUndefined } from '#lib/utils/zod-util.ts';
	import ChevronDownIcon from '~icons/solar/alt-arrow-down-linear';
	import InfoIcon from '~icons/solar/info-circle-bold-duotone';
	import Trash2Icon from '~icons/solar/trash-bin-minimalistic-bold';
	import { untrack } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod/v4';
	import { expirations } from './login-code-dialog.svelte';

	let { open = $bindable() }: { open: boolean } = $props();

	const userService = new UserService();
	const appConfigService = new AppConfigService();
	const formId = $props.id();

	type View = 'list' | 'create' | 'created';
	let view = $state<View>('list');
	let tokens = $state<SignupToken[]>();
	let created = $state<SignupToken>();
	let expandedId = $state<string | null>(null);
	let isLoading = $state(false);
	let smtpConfigured = $state(false);

	const DAY = 60 * 60 * 24;
	const formSchema = z.object({
		ttl: z.number(),
		usageLimit: z.number().min(1).max(100),
		userGroupIds: z.array(z.string()),
		email: emptyToUndefined(z.email())
	});
	const formStore = createForm(formSchema, {
		ttl: DAY,
		usageLimit: 1,
		userGroupIds: [],
		email: ''
	});
	const { inputs } = formStore;

	const titles: Record<View, () => string> = {
		list: m.signup_tokens,
		create: m.new_signup_token,
		created: m.signup_token_created
	};

	const link = (token: string) => `${page.url.origin}/st/${token}`;
	const short = (token: string) => `${token.slice(0, 3)}…${token.slice(-4)}`;
	const formatDate = (date: string | Date) =>
		new Date(date).toLocaleString(getLocale(), { dateStyle: 'medium', timeStyle: 'short' });
	const formatDay = (date: string | Date) =>
		new Date(date).toLocaleDateString(getLocale(), { dateStyle: 'medium' });

	const groupNames = (token: SignupToken) =>
		token.userGroups.map((g) => g.friendlyName).join(', ') || m.no_groups();

	function statusOf(token: SignupToken) {
		if (token.usageCount >= token.usageLimit) return { active: false, label: m.used_up() };
		if (new Date(token.expiresAt) <= new Date()) return { active: false, label: m.expired() };
		return { active: true, label: m.active() };
	}

	async function loadTokens() {
		try {
			const res = await userService.listSignupTokens({
				pagination: { page: 1, limit: 100 },
				sort: { column: 'createdAt', direction: 'desc' }
			});
			tokens = res.data;
		} catch (e) {
			axiosErrorToast(e);
		}
	}

	// New tokens start with the configured default groups, so they get the same groups as open sign-ups
	// The link can only be emailed once SMTP is set up
	async function showCreate() {
		formStore.reset();
		view = 'create';
		try {
			const config = await appConfigService.list(true);
			formStore.setValue('userGroupIds', config.signupDefaultUserGroupIDs ?? []);
			smtpConfigured = !!config.smtpHost;
		} catch (e) {
			axiosErrorToast(e);
		}
	}

	$effect(() => {
		if (!open) return;
		untrack(() => {
			tokens = undefined;
			expandedId = null;
			view = 'list';
			loadTokens();
		});
	});

	async function onSubmit() {
		const data = formStore.validate();
		if (!data) return;
		isLoading = true;
		try {
			const token = await userService.createSignupToken(
				data.ttl,
				data.usageLimit,
				data.userGroupIds,
				data.email
			);
			await loadTokens();

			// An emailed link goes straight back to the list, like an emailed login code closes its dialog
			if (data.email) {
				toast.success(m.signup_link_sent_to_email({ email: data.email }));
				view = 'list';
				return;
			}
			created = tokens?.find((t) => t.token === token);
			view = 'created';
		} catch (e) {
			axiosErrorToast(e);
		} finally {
			isLoading = false;
		}
	}

	function deleteToken(token: SignupToken) {
		openConfirmDialog({
			title: m.delete_this_signup_token(),
			message: m.delete_signup_token_consequence({ token: short(token.token) }),
			confirm: {
				label: m.delete_token(),
				destructive: true,
				action: async () => {
					try {
						await userService.deleteSignupToken(token.id);
						toast.success(m.signup_token_deleted());
						await loadTokens();
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}
</script>

<Sheet.Root bind:open>
	<Sheet.Content title={titles[view]()}>
		{#if view === 'list'}
			<p class="text-control text-fg-muted">
				{m.signup_tokens_description()}
			</p>
			<div class="overflow-hidden rounded-md border">
				{#if !tokens}
					{#each [0, 1] as i (i)}
						<div class="flex h-(--row-2line) items-center border-b px-4 last:border-b-0">
							<Skeleton class="h-3.5 w-2/5" />
						</div>
					{/each}
				{:else if tokens.length === 0}
					<p class="text-control text-fg-muted flex min-h-(--row-2line) items-center px-4">
						{m.no_signup_tokens()}
					</p>
				{:else}
					{#each tokens as token (token.id)}
						{@const status = statusOf(token)}
						{@const open = expandedId === token.id}
						<div class="border-b last:border-b-0">
							<button
								type="button"
								aria-expanded={open}
								aria-controls={`signup-token-${token.id}`}
								class="hover:bg-fill-row-hover focus-visible:bg-fill-row-hover focus-visible:outline-ring flex min-h-15 w-full items-center gap-3 px-4 py-2.5 text-start transition-colors duration-(--dur-fast) focus-visible:outline-2 focus-visible:-outline-offset-2"
								onclick={() => (expandedId = open ? null : token.id)}
							>
								<span class="flex min-w-0 flex-1 flex-col gap-0.5">
									<!-- An emailed token is easier to recognize by its recipient than by its value -->
									{#if token.sentTo}
										<span class="text-control text-fg truncate font-medium">{token.sentTo}</span>
									{:else}
										<span class="text-mono text-fg font-mono font-medium">{short(token.token)}</span
										>
									{/if}
									<span class="text-caption text-fg-muted truncate">
										<!-- Phones move the status into this line, so the row keeps its width for the text -->
										<span class="sm:hidden">{status.label} ·</span>
										{m.usage_of_limit_used({ used: token.usageCount, limit: token.usageLimit })}
										·
										{status.label === m.expired()
											? m.expired_on_date({ date: formatDay(token.expiresAt) })
											: m.expires_on_date({ date: formatDay(token.expiresAt) })}
									</span>
								</span>
								<Badge variant={status.active ? 'success' : 'muted'} class="max-sm:hidden">
									{status.label}
								</Badge>
								<ChevronDownIcon
									class={cn(
										'text-fg-muted size-3 shrink-0 transition-transform duration-(--dur-base)',
										open && 'rotate-180'
									)}
								/>
							</button>
							{#if open}
								<div
									id={`signup-token-${token.id}`}
									class="transition-motion flex flex-col items-start gap-4 px-4 pt-1 pb-4 duration-(--dur-base) starting:-translate-y-1 starting:opacity-0"
								>
									{#if status.active}
										<CopyField
											label={m.signup_link()}
											value={link(token.token)}
											copyLabel={m.copy_link()}
											truncate
										/>
									{/if}
									{@render details(token)}
									<Button
										variant="ghost"
										size="sm"
										class="-ml-3"
										aria-label={m.delete_signup_token_name({ token: short(token.token) })}
										onclick={() => deleteToken(token)}
									>
										<Trash2Icon />
										{m.delete_token()}
									</Button>
								</div>
							{/if}
						</div>
					{/each}
				{/if}
			</div>
		{:else if view === 'create'}
			<form id={formId} onsubmit={preventDefault(onSubmit)} class="flex flex-col gap-6" novalidate>
				{#if smtpConfigured}
					<FormInput
						label={m.email()}
						optional
						description={m.signup_link_email_description()}
						type="email"
						bind:input={$inputs.email}
					/>
				{/if}
				<FormInput labelFor="token-expiration" label={m.expires_after()} input={$inputs.ttl}>
					<Select.Root
						type="single"
						value={String($inputs.ttl.value)}
						onValueChange={(v) => v && formStore.setValue('ttl', Number(v))}
					>
						<Select.Trigger id="token-expiration" class="w-full">
							{expirations().find((e) => e.seconds === $inputs.ttl.value)?.label}
						</Select.Trigger>
						<Select.Content>
							{#each expirations() as expiration (expiration.seconds)}
								<Select.Item value={String(expiration.seconds)}>{expiration.label}</Select.Item>
							{/each}
						</Select.Content>
					</Select.Root>
				</FormInput>
				<FormInput labelFor="usage-limit" label={m.usage_limit()} input={$inputs.usageLimit}>
					<Input
						id="usage-limit"
						type="number"
						min={1}
						max={100}
						bind:value={
							() => $inputs.usageLimit.value,
							(value) => {
								$inputs.usageLimit.value = value;
								$inputs.usageLimit.error = null;
							}
						}
						aria-invalid={!!$inputs.usageLimit.error}
					/>
				</FormInput>
				<FormInput
					labelFor="default-groups"
					label={m.groups()}
					description={m.signup_token_groups_description()}
				>
					<UserGroupInput bind:selectedGroupIds={$inputs.userGroupIds.value} />
				</FormInput>
			</form>
		{:else if created}
			<div class="flex flex-col gap-5 sm:flex-row sm:items-start">
				<Qrcode
					value={link(created.token)}
					size={140}
					class="shrink-0 self-center rounded-sm bg-white p-2 sm:self-start"
				/>
				<CopyField
					label={m.signup_link()}
					value={link(created.token)}
					copyLabel={m.copy_link()}
					testId="signup-token-link"
					truncate
				/>
			</div>
			{@render details(created)}
			<p class="text-caption text-fg-muted flex items-start gap-1.5">
				<InfoIcon class="mt-[3px] size-3 shrink-0" />
				{m.signup_link_consequence()}
			</p>
		{/if}
		{#snippet footer()}
			{#if view === 'list'}
				<Button variant="ghost" onclick={() => (open = false)}>{m.close()}</Button>
				<Button onclick={showCreate}>{m.create_signup_token()}</Button>
			{:else if view === 'create'}
				<Button variant="ghost" onclick={() => (view = 'list')}>{m.back()}</Button>
				<Button type="submit" form={formId} {isLoading}>
					{$inputs.email?.value ? m.send_signup_link() : m.create_signup_token()}
				</Button>
			{:else}
				<Button onclick={() => (view = 'list')}>{m.done()}</Button>
			{/if}
		{/snippet}
	</Sheet.Content>
</Sheet.Root>

{#snippet details(token: SignupToken)}
	<dl class="text-control grid grid-cols-[auto_1fr] gap-x-6 gap-y-2">
		{#if token.sentTo}
			<dt class="text-fg-muted">{m.sent_to()}</dt>
			<dd class="wrap-anywhere">{token.sentTo}</dd>
		{/if}
		<dt class="text-fg-muted">{m.usage()}</dt>
		<dd>{m.usage_of_limit_used({ used: token.usageCount, limit: token.usageLimit })}</dd>
		<dt class="text-fg-muted">{m.expires()}</dt>
		<dd>{formatDate(token.expiresAt)}</dd>
		<dt class="text-fg-muted">{m.created()}</dt>
		<dd>{formatDate(token.createdAt)}</dd>
		<dt class="text-fg-muted">{m.groups()}</dt>
		<dd>{groupNames(token)}</dd>
	</dl>
{/snippet}
