<script lang="ts">
	import CopyToClipboard from '$lib/components/copy-to-clipboard.svelte';
	import MultiSelect from '$lib/components/form/multi-select.svelte';
	import SearchableSelect from '$lib/components/form/searchable-select.svelte';
	import * as Alert from '$lib/components/ui/alert';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import Label from '$lib/components/ui/label/label.svelte';
	import * as Tabs from '$lib/components/ui/tabs';
	import { m } from '$lib/paraglide/messages';
	import OidcService from '$lib/services/oidc-service';
	import UserService from '$lib/services/user-service';
	import type { User } from '$lib/types/user.type';
	import { debounced } from '$lib/utils/debounce-util';
	import { axiosErrorToast, getAxiosErrorMessage } from '$lib/utils/error-util';
	import { LucideAlertTriangle } from '@lucide/svelte';
	import { onMount } from 'svelte';

	let {
		open = $bindable(),
		clientId
	}: {
		open: boolean;
		clientId: string;
	} = $props();

	const oidcService = new OidcService();
	const userService = new UserService();

	let previewData = $state<{
		idToken?: any;
		accessToken?: any;
		userInfo?: any;
	} | null>(null);
	let loadingPreview = $state(false);
	let user: User | null = $state(null);
	let users: User[] = $state([]);
	let scopes: string[] = $state(['openid', 'email', 'profile']);
	let errorMessage: string | null = $state(null);

	async function loadPreviewData() {
		errorMessage = null;

		try {
			previewData = await oidcService.getClientPreview(clientId, user!.id, scopes.join(' '));
		} catch (e) {
			const error = getAxiosErrorMessage(e);
			errorMessage = error;
			axiosErrorToast(e);
			previewData = null;
		} finally {
			loadingPreview = false;
		}
	}

	async function loadUsers(search?: string) {
		users = (
			await userService.list({
				search,
				pagination: { limit: 3, page: 1 }
			})
		).data;
		if (!user) {
			user = users[0];
		}
	}

	async function onOpenChange(open: boolean) {
		if (!open) {
			previewData = null;
			errorMessage = null;
		} else {
			loadingPreview = true;
			await loadPreviewData().finally(() => {
				loadingPreview = false;
			});
		}
	}

	const onUserSearch = debounced(async (search: string) => await loadUsers(search), 300);

	$effect(() => {
		if (open) {
			loadPreviewData();
		}
	});

	onMount(() => {
		loadUsers();
	});
</script>

<Dialog.Root bind:open {onOpenChange}>
	<Dialog.Content class="sm-min-w[500px] max-h-[90vh] min-w-[90vw] overflow-auto lg:min-w-[1000px]">
		<Dialog.Header>
			<Dialog.Title>{m.oidc_data_preview()}</Dialog.Title>
			<Dialog.Description>
				{#if user}
					{m.preview_for_user({ name: user.firstName + ' ' + user.lastName, email: user.email })}
				{:else}
					{m.preview_the_oidc_data_that_would_be_sent_for_this_user()}
				{/if}
			</Dialog.Description>
		</Dialog.Header>

		<div class="overflow-auto px-4">
			{#if loadingPreview}
				<div class="flex items-center justify-center py-12">
					<div class="h-8 w-8 animate-spin rounded-full border-b-2 border-gray-900"></div>
				</div>
			{/if}

			<div class="flex justify-start gap-3">
				<div>
					<Label class="text-sm font-medium">Users</Label>
					<div>
						<SearchableSelect
							class="w-48"
							items={[
								{ value: '', label: m.all_users() },
								...Object.values(users).map((user) => ({
									value: user.id,
									label: user.username
								}))
							]}
							value={user?.id || ''}
							oninput={(e) => onUserSearch(e.currentTarget.value)}
							onSelect={(value) => {
								user = users.find((u) => u.id === value) || null;
								loadPreviewData();
							}}
						/>
					</div>
				</div>
				<div>
					<Label class="text-sm font-medium">Scopes</Label>
					<MultiSelect
						items={[
							{ value: 'openid', label: 'openid' },
							{ value: 'email', label: 'email' },
							{ value: 'profile', label: 'profile' },
							{ value: 'groups', label: 'groups' }
						]}
						bind:selectedItems={scopes}
					/>
				</div>
			</div>

			{#if errorMessage && !loadingPreview}
				<Alert.Root variant="destructive" class="mb-6">
					<LucideAlertTriangle class="h-4 w-4" />
					<Alert.Title>{m.access_denied()}</Alert.Title>
					<Alert.Description>
						{errorMessage}
					</Alert.Description>
				</Alert.Root>
			{/if}

			{#if previewData && !loadingPreview}
				<Tabs.Root value="id-token" class="mt-5 w-full">
					<Tabs.List class="mb-6 grid w-full grid-cols-3">
						<Tabs.Trigger value="id-token">{m.id_token()}</Tabs.Trigger>
						<Tabs.Trigger value="access-token">{m.access_token()}</Tabs.Trigger>
						<Tabs.Trigger value="userinfo">{m.userinfo_response()}</Tabs.Trigger>
					</Tabs.List>
					<Tabs.Content value="id-token">
						{@render tabContent(previewData.idToken, m.id_token_payload())}
					</Tabs.Content>

					<Tabs.Content value="access-token" class="mt-4">
						{@render tabContent(previewData.accessToken, m.access_token_payload())}
					</Tabs.Content>

					<Tabs.Content value="userinfo" class="mt-4">
						{@render tabContent(previewData.userInfo, m.userinfo_endpoint_response())}
					</Tabs.Content>
				</Tabs.Root>
			{/if}
		</div>
	</Dialog.Content>
</Dialog.Root>

{#snippet tabContent(data: any, title: string)}
	<div class="space-y-4">
		<div class="mb-6 flex items-center justify-between">
			<Label class="text-lg font-semibold">{title}</Label>
			<CopyToClipboard value={JSON.stringify(data, null, 2)}>
				<Button size="sm" variant="outline">{m.copy_all()}</Button>
			</CopyToClipboard>
		</div>
		<div class="space-y-3">
			{#each Object.entries(data || {}) as [key, value]}
				<div class="grid grid-cols-1 items-start gap-4 border-b pb-3 md:grid-cols-[200px_1fr]">
					<Label class="pt-1 text-sm font-medium">{key}</Label>
					<div class="min-w-0">
						<CopyToClipboard value={typeof value === 'string' ? value : JSON.stringify(value)}>
							<div
								class="text-muted-foreground bg-muted/30 hover:bg-muted/50 cursor-pointer rounded px-3 py-2 font-mono text-sm"
							>
								{typeof value === 'object' ? JSON.stringify(value, null, 2) : value}
							</div>
						</CopyToClipboard>
					</div>
				</div>
			{/each}
		</div>
	</div>
{/snippet}
