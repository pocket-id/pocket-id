<script lang="ts">
	import { page } from '$app/state';
	import { openConfirmDialog } from '$lib/components/confirm-dialog/';
	import AdvancedTable from '$lib/components/table/advanced-table.svelte';
	import { Badge, type BadgeVariant } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import { m } from '$lib/paraglide/messages';
	import UserService from '$lib/services/user-service';
	import type {
		AdvancedTableColumn,
		CreateAdvancedTableActions
	} from '$lib/types/advanced-table.type';
	import type { SignupTokenDto } from '$lib/types/signup-token.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { Copy, Trash2 } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';

	let {
		open = $bindable()
	}: {
		open: boolean;
	} = $props();

	const userService = new UserService();
	let tableRef: AdvancedTable<SignupTokenDto>;

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
						await tableRef.refresh();
						toast.success(m.signup_token_deleted_successfully());
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

	const columns: AdvancedTableColumn<SignupTokenDto>[] = [
		{ label: m.token(), column: 'token', cell: TokenCell },
		{ label: m.status(), key: 'status', cell: StatusCell },
		{
			label: m.usage(),
			column: 'usageCount',
			sortable: true,
			cell: UsageCell
		},
		{
			label: m.expires(),
			column: 'expiresAt',
			sortable: true,
			value: (item) => formatDate(item.expiresAt)
		},
		{ label: 'Usage Limit', column: 'usageLimit' },
		{
			label: m.created(),
			column: 'createdAt',
			sortable: true,
			hidden: true,
			value: (item) => formatDate(item.createdAt)
		}
	];

	const actions: CreateAdvancedTableActions<SignupTokenDto> = (_) => [
		{
			label: m.copy(),
			icon: Copy,
			onClick: (token) => copySignupLink(token)
		},
		{
			label: m.delete(),
			icon: Trash2,
			onClick: (token) => deleteToken(token)
		}
	];
</script>

{#snippet TokenCell({ item }: { item: SignupTokenDto })}
	<span class="font-mono text-xs">
		{item.token.substring(0, 3)}...{item.token.substring(Math.max(item.token.length - 4, 0))}
	</span>
{/snippet}

{#snippet StatusCell({ item }: { item: SignupTokenDto })}
	{@const status = getTokenStatus(item)}
	{@const statusBadge = getStatusBadge(status)}
	<Badge class="rounded-full" variant={statusBadge.variant}>
		{statusBadge.text}
	</Badge>
{/snippet}

{#snippet UsageCell({ item }: { item: SignupTokenDto })}
	<div class="flex items-center gap-1">
		{item.usageCount}
		{m.of()}
		{item.usageLimit}
	</div>
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
			<AdvancedTable
				id="signup-token-list"
				withoutSearch={true}
				fetchCallback={userService.listSignupTokens}
				bind:this={tableRef}
				{columns}
				{actions}
			/>
		</div>
		<Dialog.Footer class="mt-3">
			<Button onclick={() => (open = false)}>
				{m.close()}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
