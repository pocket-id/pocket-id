<script lang="ts">
	import CopyButton from '#lib/components/copy-button.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import type { DeviceLoginRequest } from '#lib/types/device-login.type.ts';
	import { cn } from '#lib/utils/style.ts';
	import Qrcode from './qrcode.svelte';

	let {
		request,
		ended = false
	}: {
		request: DeviceLoginRequest;
		// The server can end the request before the local countdown does, and a denied code is just as dead
		ended?: boolean;
	} = $props();

	let showQr = $state(false);
	let now = $state(Date.now());
	$effect(() => {
		const timer = setInterval(() => (now = Date.now()), 1000);
		return () => clearInterval(timer);
	});

	const secondsLeft = $derived(
		Math.max(0, Math.ceil((new Date(request.expiresAt).getTime() - now) / 1000))
	);
	const expired = $derived(ended || secondsLeft === 0);
	const countdown = $derived(
		`${Math.floor(secondsLeft / 60)}:${String(secondsLeft % 60).padStart(2, '0')}`
	);
	const codeGroups = $derived([request.userCode.slice(0, 4), request.userCode.slice(4)]);
</script>

<div class="w-full rounded-md border p-4">
	<div class="flex flex-col items-center gap-4">
		<!-- A phone cannot scan its own screen, so on phones the code comes first and the QR waits behind a text action -->
		<Qrcode
			value={request.verificationUriComplete}
			size={144}
			role="img"
			aria-label={m.device_login_qr_code()}
			aria-hidden={expired}
			class={cn(
				'transition-opacity duration-(--dur-base) max-sm:order-last',
				expired && 'opacity-20',
				!showQr && 'max-sm:hidden'
			)}
		/>
		<div class="flex w-full min-w-0 flex-col gap-3">
			<p class="text-caption text-fg-muted text-center wrap-anywhere">
				{m.visit_and_enter({ url: request.verificationUri })}
			</p>
			<div
				class={cn('flex items-center font-mono text-xl font-medium', expired && 'text-fg-muted')}
				data-testid="device-login-code"
			>
				{#each codeGroups as group, i (i)}
					{#if i > 0}
						<span class="text-fg-muted w-7 shrink-0 text-center text-xs">–</span>
					{/if}
					<span class="bg-surface-sunken flex min-w-0 flex-1 rounded-sm">
						{#each group as char, j (j)}
							<span class="grid h-11 min-w-0 flex-1 place-items-center">{char}</span>
						{/each}
					</span>
				{/each}
			</div>
			<CopyButton
				labeled
				value={request.userCode}
				label={m.copy_code()}
				class={cn('self-center', expired && 'invisible')}
			/>
		</div>
		{#if !expired}
			<Button
				variant="link"
				class="h-11 sm:hidden"
				aria-expanded={showQr}
				onclick={() => (showQr = !showQr)}
			>
				{showQr ? m.hide_qr_code() : m.show_qr_code()}
			</Button>
		{/if}
	</div>
	<p class="text-caption text-fg-muted mt-3 text-center tabular-nums">
		{expired ? m.code_expired() : m.expires_in_countdown({ time: countdown })}
	</p>
</div>
