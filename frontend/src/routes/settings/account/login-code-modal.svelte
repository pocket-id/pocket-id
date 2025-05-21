<script lang="ts">
	import { page } from '$app/state';
	import CopyToClipboard from '$lib/components/copy-to-clipboard.svelte';
	import Qrcode from '$lib/components/qrcode/qrcode.svelte';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Separator } from '$lib/components/ui/separator';
	import { m } from '$lib/paraglide/messages';
	import UserService from '$lib/services/user-service';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { mode } from 'mode-watcher';

	let {
		show = $bindable()
	}: {
		show: boolean;
	} = $props();

	const userService = new UserService();

	let code: string | null = $state(null);
	let loginCodeLink: string | null = $state(null);

	$effect(() => {
		if (show) {
			const expiration = new Date(Date.now() + 15 * 60 * 1000);
			userService
				.createOneTimeAccessToken(expiration, 'me')
				.then((c) => {
					code = c;
					loginCodeLink = page.url.origin + '/lc/' + code;
				})
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
			<Dialog.Title>{m.login_code()}</Dialog.Title>
			<Dialog.Description
				>{m.sign_in_using_the_following_code_the_code_will_expire_in_minutes()}
			</Dialog.Description>
		</Dialog.Header>

		<div class="flex flex-col items-center gap-2">
			<CopyToClipboard value={code!}>
				<p class="text-3xl font-semibold">{code}</p>
			</CopyToClipboard>
			<div class="text-muted-foreground my-2 flex items-center justify-center gap-3">
				<Separator />
				<p class="text-xs text-nowrap">{m.or_visit()}</p>
				<Separator />
			</div>

			<Qrcode
				class="mb-2"
				value={loginCodeLink}
				size={180}
				color={mode.current === 'dark' ? '#FFFFFF' : '#000000'}
				backgroundColor={mode.current === 'dark' ? '#000000' : '#FFFFFF'}
			/>
			<CopyToClipboard value={loginCodeLink!}>
				<p data-testId="login-code-link">{loginCodeLink!}</p>
			</CopyToClipboard>
		</div>
	</Dialog.Content>
</Dialog.Root>
