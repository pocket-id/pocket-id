<script lang="ts">
	import { page } from '$app/state';
	import { openConfirmDialog } from '$lib/components/confirm-dialog/';
	import { Badge, type BadgeVariant } from '$lib/components/ui/badge';
	import { Button, buttonVariants } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import { m } from '$lib/paraglide/messages';
	import UserService from '$lib/services/user-service';
	import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
	import type { SignupTokenDto } from '$lib/types/signup-token.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { Copy, Ellipsis, Trash2 } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import PocketIdTable from '$lib/components/pocket-id-table/pocket-id-table.svelte';
	import type { ColumnSpec } from '$lib/components/pocket-id-table';

	let {
		open = $bindable(),
		signupTokens = $bindable(),
		signupTokensRequestOptions,
		onTokenDeleted
	}: {
		open: boolean;
		signupTokens: Paginated<SignupTokenDto>;
		signupTokensRequestOptions: SearchPaginationSortRequest;
		onTokenDeleted?: () => Promise<void>;
	} = $props();

	const userService = new UserService();

	function formatDate(dateStr: string | undefined) {
		if (!dateStr) return m.never();
		return new Date(dateStr).toLocaleString();
	}

	async function deleteToken(token: SignupTokenDto) {
		openConfirmDialog({
			title: m.delete_signup_token(),
			message: m.are_you_sure_you_want_to_delete_this_signup_token(),
			confirm: {
				label: m.delete(),
				destructive: true,
				action: async () => {
					try {
						await userService.deleteSignupToken(token.id);
						toast.success(m.signup_token_deleted_successfully());

						// Refresh the tokens
						if (onTokenDeleted) {
							await onTokenDeleted();
						}
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}

	function onOpenChange(isOpen: boolean) {
		open = isOpen;
	}

	function isTokenExpired(expiresAt: string) {
		return new Date(expiresAt) < new Date();
	}

	function isTokenUsedUp(token: SignupTokenDto) {
		return token.usageCount >= token.usageLimit;
	}

	function getTokenStatus(token: SignupTokenDto) {
		if (isTokenExpired(token.expiresAt)) return 'expired';
		if (isTokenUsedUp(token)) return 'used-up';
		return 'active';
	}

	function getStatusBadge(status: string): { variant: BadgeVariant; text: string } {
		switch (status) {
			case 'expired':
				return { variant: 'destructive', text: m.expired() };
			case 'used-up':
				return { variant: 'secondary', text: m.used_up() };
			default:
				return { variant: 'default', text: m.active() };
		}
	}

	function copySignupLink(token: SignupTokenDto) {
		const signupLink = `${page.url.origin}/st/${token.token}`;
		navigator.clipboard
			.writeText(signupLink)
			.then(() => {
				toast.success(m.copied());
			})
			.catch((err) => {
				axiosErrorToast(err);
			});
	}

	const columns = [
		{ title: m.token(), cell: TokenCell },
		{ title: m.status(), cell: StatusBadgeCell },
		{ title: m.usage(), accessorKey: 'usageCount', sortable: true, cell: UsageCountCell },
		{ title: m.expires(), accessorKey: 'expiresAt', sortable: true, cell: ExpiresAtCell },
		{ title: m.created(), accessorKey: 'createdAt', sortable: true, cell: CreatedAtCell },
		{ title: m.actions(), hidden: true }
	] satisfies ColumnSpec<SignupTokenDto>[];
</script>

{#snippet TokenCell({ item }: { item: SignupTokenDto })}
	{item.token.substring(0, 2)}...{item.token.substring(item.token.length - 4)}
{/snippet}

{#snippet StatusBadgeCell({ item }: { item: SignupTokenDto })}
	{@const status = getTokenStatus(item)}
	{@const statusBadge = getStatusBadge(status)}
	<Badge class="rounded-full" variant={statusBadge.variant}>
		{statusBadge.text}
	</Badge>
{/snippet}

{#snippet UsageCountCell({ item }: { item: SignupTokenDto })}
	<div class="flex items-center gap-1">
		{`${item.usageCount} ${m.of()} ${item.usageLimit}`}
	</div>
{/snippet}

{#snippet ExpiresAtCell({ item }: { item: SignupTokenDto })}
	<div class="flex items-center gap-1">
		{formatDate(item.expiresAt)}
	</div>
{/snippet}

{#snippet CreatedAtCell({ item }: { item: SignupTokenDto })}
	<div class="flex items-center gap-1">
		{formatDate(item.createdAt)}
	</div>
{/snippet}

{#snippet RowActions({ item }: { item: SignupTokenDto })}
	<DropdownMenu.Root>
		<DropdownMenu.Trigger class={buttonVariants({ variant: 'ghost', size: 'icon' })}>
			<Ellipsis class="size-4" />
			<span class="sr-only">{m.toggle_menu()}</span>
		</DropdownMenu.Trigger>
		<DropdownMenu.Content align="end">
			<DropdownMenu.Item onclick={() => copySignupLink(item)}>
				<Copy class="mr-2 size-4" />
				{m.copy()}
			</DropdownMenu.Item>
			<DropdownMenu.Item class="text-red-500 focus:!text-red-700" onclick={() => deleteToken(item)}>
				<Trash2 class="mr-2 size-4" />
				{m.delete()}
			</DropdownMenu.Item>
		</DropdownMenu.Content>
	</DropdownMenu.Root>
{/snippet}

<Dialog.Root {open} {onOpenChange}>
	<Dialog.Content class="sm-min-w[500px] max-h-[90vh] min-w-[90vw] overflow-auto lg:min-w-[1000px]">
		<Dialog.Header>
			<Dialog.Title>{m.manage_signup_tokens()}</Dialog.Title>
			<Dialog.Description>
				{m.view_and_manage_active_signup_tokens()}
			</Dialog.Description>
		</Dialog.Header>

		<div class="flex-1 overflow-hidden">
			<PocketIdTable
				items={signupTokens}
				bind:requestOptions={signupTokensRequestOptions}
				onRefresh={async (opts) => (signupTokens = await userService.listSignupTokens(opts))}
				{columns}
				persistKey="pocket-id-signup-tokens"
				rowActions={RowActions}
				withoutSearch
			/>
		</div>
		<Dialog.Footer class="mt-3">
			<Button onclick={() => (open = false)}>
				{m.close()}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
