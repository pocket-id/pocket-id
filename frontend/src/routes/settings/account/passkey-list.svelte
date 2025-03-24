<script lang="ts">
	import { openConfirmDialog } from '$lib/components/confirm-dialog/';
	import { Button } from '$lib/components/ui/button';
	import { Separator } from '$lib/components/ui/separator';
	import { Badge } from '$lib/components/ui/badge';
	import { Tooltip, TooltipContent, TooltipTrigger } from '$lib/components/ui/tooltip';
	import WebauthnService from '$lib/services/webauthn-service';
	import type { Passkey } from '$lib/types/passkey.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideKeyRound, LucidePencil, LucideTrash, LucideCalendar } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';
	import RenamePasskeyModal from './rename-passkey-modal.svelte';
	import { m } from '$lib/paraglide/messages';

	let { passkeys = $bindable() }: { passkeys: Passkey[] } = $props();

	const webauthnService = new WebauthnService();

	let passkeyToRename: Passkey | null = $state(null);

	async function deletePasskey(passkey: Passkey) {
		openConfirmDialog({
			title: m.delete_passkey_name({ passkeyName: passkey.name }),
			message: m.are_you_sure_you_want_to_delete_this_passkey(),
			confirm: {
				label: m.delete(),
				destructive: true,
				action: async () => {
					try {
						await webauthnService.removeCredential(passkey.id);
						passkeys = await webauthnService.listCredentials();
						toast.success(m.passkey_deleted_successfully());
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}

	// Function to determine if a passkey was recently added (within the last 7 days)
	function isRecentlyAdded(date: string): boolean {
		const createdDate = new Date(date);
		const currentDate = new Date();
		const differenceInDays = Math.floor(
			(currentDate.getTime() - createdDate.getTime()) / (1000 * 3600 * 24)
		);
		return differenceInDays <= 7;
	}
</script>

<div class="space-y-3">
	{#each passkeys as passkey, i}
		<div class="bg-card hover:bg-muted/50 group rounded-lg p-3 transition-colors">
			<div class="flex items-center justify-between">
				<div class="flex items-start gap-3">
					<div class="bg-primary/10 text-primary mt-1 rounded-lg p-2">
						<LucideKeyRound class="h-5 w-5" />
					</div>
					<div>
						<div class="flex items-center gap-2">
							<p class="font-medium">{passkey.name}</p>
							{#if isRecentlyAdded(passkey.createdAt)}
								<Badge variant="outline" class="bg-primary/10 text-primary text-xs">New</Badge>
							{/if}
						</div>
						<div class="text-muted-foreground mt-1 flex items-center text-xs">
							<LucideCalendar class="mr-1 h-3 w-3" />
							{m.added_on()}
							{new Date(passkey.createdAt).toLocaleDateString()}
						</div>
					</div>
				</div>

				<div class="flex items-center gap-2 opacity-0 transition-opacity group-hover:opacity-100">
					<Tooltip>
						<TooltipTrigger asChild>
							<Button
								on:click={() => (passkeyToRename = passkey)}
								size="icon"
								variant="ghost"
								class="h-8 w-8"
								aria-label={m.rename()}
							>
								<LucidePencil class="h-4 w-4" />
							</Button>
						</TooltipTrigger>
						<TooltipContent>{m.rename()}</TooltipContent>
					</Tooltip>

					<Tooltip>
						<TooltipTrigger asChild>
							<Button
								on:click={() => deletePasskey(passkey)}
								size="icon"
								variant="ghost"
								class="hover:bg-destructive/10 hover:text-destructive h-8 w-8"
								aria-label={m.delete()}
							>
								<LucideTrash class="h-4 w-4" />
							</Button>
						</TooltipTrigger>
						<TooltipContent>{m.delete()}</TooltipContent>
					</Tooltip>
				</div>
			</div>
		</div>
	{/each}
</div>

<RenamePasskeyModal
	bind:passkey={passkeyToRename}
	callback={async () => (passkeys = await webauthnService.listCredentials())}
/>
