<script lang="ts">
	import { page } from '$app/state';
	import CopyToClipboard from '$lib/components/copy-to-clipboard.svelte';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Separator } from '$lib/components/ui/separator';
	import UserService from '$lib/services/user-service';
	import { axiosErrorToast } from '$lib/utils/error-util';

	let {
		show = $bindable()
	}: {
		show: boolean;
	} = $props();

	const userService = new UserService();

	let code: string | null = $state(null);

	$effect(() => {
		if (show) {
			const expiration = new Date(Date.now() + 15 * 60 * 1000);
			userService
				.createOneTimeAccessToken(expiration, 'me')
				.then((c) => (code = c))
				.catch((e) => axiosErrorToast(e));
		}
	});

	function onOpenChange(open: boolean) {
		if (!open) {
			code = null;
			show = false;
		}
	}
</script>

<Dialog.Root open={!!code} {onOpenChange}>
	<Dialog.Content class="max-w-md">
		<Dialog.Header>
			<Dialog.Title>Login Code</Dialog.Title>
			<Dialog.Description
				>Sign in using the following code. The code will expire in 15 minutes.
			</Dialog.Description>
		</Dialog.Header>

		<div class="flex flex-col items-center gap-2">
			<CopyToClipboard value={code!}>
				<p class="text-3xl font-semibold">{code}</p>
			</CopyToClipboard>
			<div class="text-muted-foreground flex items-center justify-center gap-3">
				<Separator />
				<p class="text-nowrap text-xs">or visit</p>
				<Separator />
			</div>
			<div>
				<CopyToClipboard value={page.url.origin + '/lc/' + code!}>
					<p data-testId="login-code-link">{page.url.origin + '/lc/' + code!}</p>
				</CopyToClipboard>
			</div>
		</div>
	</Dialog.Content>
</Dialog.Root>
