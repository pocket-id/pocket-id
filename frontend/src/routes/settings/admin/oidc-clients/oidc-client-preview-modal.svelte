<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog';
	import * as Tabs from '$lib/components/ui/tabs';
	import { Button } from '$lib/components/ui/button';
	import Label from '$lib/components/ui/label/label.svelte';
	import CopyToClipboard from '$lib/components/copy-to-clipboard.svelte';
	import * as Alert from '$lib/components/ui/alert';
	import { m } from '$lib/paraglide/messages';
	import OidcService from '$lib/services/oidc-service';
	import UserService from '$lib/services/user-service';
	import { axiosErrorToast, getAxiosErrorMessage } from '$lib/utils/error-util';
	import type { User } from '$lib/types/user.type';
	import { LucideAlertTriangle } from '@lucide/svelte';

	let {
		userId = $bindable(),
		clientId
	}: {
		userId: string | null;
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
	let errorMessage: string | null = $state(null);

	async function loadPreviewData() {
		if (!userId) return;

		loadingPreview = true;
		errorMessage = null;

		try {
			const [preview, userInfo] = await Promise.all([
				oidcService.getClientPreview(clientId, userId),
				userService.get(userId)
			]);
			previewData = preview;
			user = userInfo;
		} catch (e) {
			const error = getAxiosErrorMessage(e);
			errorMessage = error;

			// Still show the toast for consistency with other parts of the app
			axiosErrorToast(e);

			// Try to get user info even if preview fails
			try {
				user = await userService.get(userId);
			} catch (userError) {
				user = null;
			}

			previewData = null;
		} finally {
			loadingPreview = false;
		}
	}

	function onOpenChange(open: boolean) {
		if (!open) {
			previewData = null;
			user = null;
			errorMessage = null;
			userId = null;
		} else if (userId) {
			loadPreviewData();
		}
	}

	$effect(() => {
		if (userId) {
			loadPreviewData();
		}
	});
</script>

<Dialog.Root open={!!userId} {onOpenChange}>
	<Dialog.Content
		class="max-h-[90vh] min-w-[800px] max-w-[95vw] overflow-auto md:min-w-[1000px] lg:min-w-[1200px]"
	>
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
				<Tabs.Root value="id-token" class="w-full">
					<Tabs.List class="mb-6 grid w-full grid-cols-3">
						<Tabs.Trigger value="id-token">{m.id_token()}</Tabs.Trigger>
						<Tabs.Trigger value="access-token">{m.access_token()}</Tabs.Trigger>
						<Tabs.Trigger value="userinfo">{m.userinfo_response()}</Tabs.Trigger>
					</Tabs.List>

					<Tabs.Content value="id-token" class="mt-4">
						<div class="space-y-4">
							<div class="mb-6 flex items-center justify-between">
								<Label class="text-lg font-semibold">{m.id_token_payload()}</Label>
								<CopyToClipboard value={JSON.stringify(previewData.idToken, null, 2)}>
									<Button size="sm" variant="outline">{m.copy_all()}</Button>
								</CopyToClipboard>
							</div>
							<div class="space-y-3">
								{#each Object.entries(previewData.idToken || {}) as [key, value]}
									<div class="grid grid-cols-[200px_1fr] items-start gap-4 border-b pb-3">
										<Label class="pt-1 text-sm font-medium">{key}</Label>
										<div class="min-w-0">
											<CopyToClipboard
												value={typeof value === 'string' ? value : JSON.stringify(value)}
											>
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
					</Tabs.Content>

					<Tabs.Content value="access-token" class="mt-4">
						<div class="space-y-4">
							<div class="mb-6 flex items-center justify-between">
								<Label class="text-lg font-semibold">{m.access_token_payload()}</Label>
								<CopyToClipboard value={JSON.stringify(previewData.accessToken, null, 2)}>
									<Button size="sm" variant="outline">{m.copy_all()}</Button>
								</CopyToClipboard>
							</div>
							<div class="space-y-3">
								{#each Object.entries(previewData.accessToken || {}) as [key, value]}
									<div class="grid grid-cols-[200px_1fr] items-start gap-4 border-b pb-3">
										<Label class="pt-1 text-sm font-medium">{key}</Label>
										<div class="min-w-0">
											<CopyToClipboard
												value={typeof value === 'string' ? value : JSON.stringify(value)}
											>
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
					</Tabs.Content>

					<Tabs.Content value="userinfo" class="mt-4">
						<div class="space-y-4">
							<div class="mb-6 flex items-center justify-between">
								<Label class="text-lg font-semibold">{m.userinfo_endpoint_response()}</Label>
								<CopyToClipboard value={JSON.stringify(previewData.userInfo, null, 2)}>
									<Button size="sm" variant="outline">{m.copy_all()}</Button>
								</CopyToClipboard>
							</div>
							<div class="space-y-3">
								{#each Object.entries(previewData.userInfo || {}) as [key, value]}
									<div class="grid grid-cols-[200px_1fr] items-start gap-4 border-b pb-3">
										<Label class="pt-1 text-sm font-medium">{key}</Label>
										<div class="min-w-0">
											<CopyToClipboard
												value={typeof value === 'string' ? value : JSON.stringify(value)}
											>
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
					</Tabs.Content>
				</Tabs.Root>
			{/if}
		</div>
	</Dialog.Content>
</Dialog.Root>
