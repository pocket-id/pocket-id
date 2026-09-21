<script lang="ts">
	import CollapsibleCard from '$lib/components/collapsible-card.svelte';
	import * as Card from '$lib/components/ui/card';
	import { m } from '$lib/paraglide/messages';
	import ApisService from '$lib/services/apis-service';
	import type { ApiCimdAccessUpdate, ApiCreate, ApiPermissionInput } from '$lib/types/api.type';
	import { trackUnsavedValue } from '$lib/utils/unsaved-changes-util.svelte';
	import { LucideChevronLeft } from '@lucide/svelte';
	import { backNavigate } from '../../users/navigate-back-util';
	import ApiForm from '../api-form.svelte';
	import ApiAccessCard from './api-access-card.svelte';
	import ApiPermissionsInput from './api-permissions-input.svelte';

	let { data } = $props();
	let api = $state(data.api);
	let permissions = $state<ApiPermissionInput[]>(toPermissionInputs(data.api.permissions));

	function toPermissionInputs(
		apiPermissions: { key: string; name: string; description?: string }[]
	): ApiPermissionInput[] {
		const inputs = apiPermissions.map((p) => ({
			key: p.key,
			name: p.name,
			description: p.description ?? ''
		}));
		// Show an empty row so the user doesn't have to add one first
		return inputs.length > 0 ? inputs : [{ key: '', name: '', description: '' }];
	}

	function isEmptyPermission(p: ApiPermissionInput) {
		return !p.key.trim() && !p.name.trim() && !p.description.trim();
	}

	const apisService = new ApisService();
	const backNavigation = backNavigate('/settings/admin/apis');

	let accessCard = $state<ApiAccessCard>();

	async function updateApi(updated: ApiCreate) {
		api = { ...api, ...(await apisService.update(api.id, { name: updated.name })) };
	}

	async function updatePermissions(updatedPermissions: ApiPermissionInput[]) {
		try {
			const res = await apisService.updatePermissions(api.id, updatedPermissions);
			api = res;
			permissions = toPermissionInputs(res.permissions);
		} finally {
			// A removed permission takes the client grants that referenced it with it
			await accessCard?.refresh();
		}
	}

	trackUnsavedValue(
		() => permissions.filter((p) => !isEmptyPermission(p)),
		(savedPermissions) => (permissions = toPermissionInputs(savedPermissions)),
		updatePermissions
	);

	async function updateCimdAccess(update: ApiCimdAccessUpdate) {
		try {
			api = await apisService.updateCimdAccess(api.id, update);
		} finally {
			await accessCard?.refresh();
		}
	}
</script>

<svelte:head>
	<title>{api.name}</title>
</svelte:head>

<div>
	<button type="button" class="text-muted-foreground flex text-sm" onclick={backNavigation.go}>
		<LucideChevronLeft class="size-5" />
		{m.back()}
	</button>
</div>

<Card.Root>
	<Card.Header>
		<Card.Title>{m.general()}</Card.Title>
	</Card.Header>
	<Card.Content>
		<ApiForm existingApi={api} callback={updateApi} />
	</Card.Content>
</Card.Root>

<CollapsibleCard
	id="api-permissions"
	title={m.api_permissions()}
	description={m.api_permissions_description()}
	defaultExpanded={true}
>
	<ApiPermissionsInput bind:permissions />
</CollapsibleCard>

<ApiAccessCard bind:this={accessCard} {api} onCimdAccessSave={updateCimdAccess} />
