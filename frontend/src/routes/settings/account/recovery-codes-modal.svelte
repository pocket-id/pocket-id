<script lang="ts">
	import CopyToClipboard from '$lib/components/copy-to-clipboard.svelte';
	import * as Alert from '$lib/components/ui/alert';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import { m } from '$lib/paraglide/messages';
	import { buildRecoveryKitPdf, downloadBlob } from '$lib/utils/recovery-code-pdf';
	import { LucideAlertTriangle, LucideDownload, LucideFileText } from '@lucide/svelte';

	let {
		show = $bindable(),
		codes
	}: {
		show: boolean;
		codes: string[];
	} = $props();

	const allCodesJoined = $derived(codes.join('\n'));

	function downloadTxt() {
		const lines = [
			m.recovery_codes_title(),
			'',
			m.recovery_codes_pdf_intro_1(),
			m.recovery_codes_pdf_intro_2(),
			'',
			...codes,
			'',
			m.recovery_codes_pdf_warning_title().toUpperCase(),
			m.recovery_codes_pdf_warning_1(),
			m.recovery_codes_pdf_warning_2()
		];
		const blob = new Blob([lines.join('\n') + '\n'], { type: 'text/plain;charset=utf-8' });
		downloadBlob(blob, 'recovery-codes.txt');
	}

	function downloadPdf() {
		const blob = buildRecoveryKitPdf({
			title: m.recovery_codes_title(),
			subtitle: m.recovery_codes_pdf_subtitle(),
			introParagraphs: [m.recovery_codes_pdf_intro_1(), m.recovery_codes_pdf_intro_2()],
			codesHeading: m.recovery_codes_pdf_codes_heading(),
			codes,
			warningTitle: m.recovery_codes_pdf_warning_title(),
			warningParagraphs: [m.recovery_codes_pdf_warning_1(), m.recovery_codes_pdf_warning_2()]
		});
		downloadBlob(blob, 'recovery-codes.pdf');
	}

	function onOpenChange(open: boolean) {
		if (!open) {
			show = false;
		}
	}
</script>

<Dialog.Root open={show} {onOpenChange}>
	<Dialog.Content class="max-w-lg" onOpenAutoFocus={(e) => e.preventDefault()}>
		<Dialog.Header>
			<Dialog.Title>{m.recovery_codes_title()}</Dialog.Title>
			<Dialog.Description>
				{m.recovery_codes_modal_description()}
			</Dialog.Description>
		</Dialog.Header>

		<Alert.Root variant="warning" class="flex gap-3">
			<LucideAlertTriangle class="size-4" />
			<div>
				<Alert.Title class="font-semibold">{m.recovery_codes_save_now_title()}</Alert.Title>
				<Alert.Description class="text-sm">
					{m.recovery_codes_save_now_description()}
				</Alert.Description>
			</div>
		</Alert.Root>

		<CopyToClipboard value={allCodesJoined}>
			<ul
				class="bg-muted/40 grid grid-cols-2 gap-x-6 gap-y-1 rounded-lg p-4 font-mono text-sm"
				aria-label={m.recovery_codes_list_label()}
			>
				{#each codes as code}
					<li class="select-all tracking-wider">{code}</li>
				{/each}
			</ul>
		</CopyToClipboard>

		<Dialog.Footer class="flex-col sm:flex-row sm:justify-between">
			<div class="flex flex-wrap gap-2">
				<Button variant="outline" onclick={downloadTxt}>
					<LucideFileText class="mr-1 size-4" />
					{m.download_txt()}
				</Button>
				<Button variant="outline" onclick={downloadPdf}>
					<LucideDownload class="mr-1 size-4" />
					{m.download_pdf()}
				</Button>
			</div>
			<Button onclick={() => onOpenChange(false)}>{m.i_saved_my_codes()}</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
