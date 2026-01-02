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
			userService
				.createOneTimeAccessToken('me')
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
	<Dialog.Content class="max-w-md" onOpenAutoFocus={(e) => e.preventDefault()}>
		<Dialog.Header>
			<Dialog.Title>{m.login_code()}</Dialog.Title>
			<Dialog.Description
				>{m.sign_in_using_the_following_code_the_code_will_expire_in_minutes()}
			</Dialog.Description>
		</Dialog.Header>

		<div class="flex flex-col items-center gap-2">
			<CopyToClipboard value={code!}>
				<p class="text-3xl font-bold">{code}</p>
			</CopyToClipboard>
			<div class="flex items-center justify-center gap-3 my-2 text-muted-foreground">
				<Separator />
				<p class="text-xs text-nowrap">{m.or_visit()}</p>
				<Separator />
			</div>

			<Qrcode
				class="mb-2"
				value={loginCodeLink}
				size={150}
				color={mode.current === 'dark' ? '#FFFFFF' : '#000000'}
				backgroundColor={mode.current === 'dark' ? '#000000' : '#FFFFFF'}
			/>
			<CopyToClipboard value={loginCodeLink!}>
				<p class="text-sm" data-testId="login-code-link">{loginCodeLink!}</p>
			</CopyToClipboard>
		</div>
	</Dialog.Content>
</Dialog.Root>
