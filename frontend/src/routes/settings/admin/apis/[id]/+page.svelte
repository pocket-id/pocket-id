<script lang="ts">
	import CollapsibleCard from '$lib/components/collapsible-card.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { m } from '$lib/paraglide/messages';
	import ApisService from '$lib/services/apis-service';
	import type { ApiCreate, ApiPermissionInput } from '$lib/types/api.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideChevronLeft } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import { backNavigate } from '../../users/navigate-back-util';
	import ApiForm from '../api-form.svelte';
	import ApiPermissionsInput from './api-permissions-input.svelte';

	let { data } = $props();
	let api = $state(data.api);
	let permissions = $state<ApiPermissionInput[]>(
		data.api.permissions.map((p) => ({
			key: p.key,
			name: p.name,
			description: p.description ?? ''
		}))
	);

	const apisService = new ApisService();
	const backNavigation = backNavigate('/settings/admin/apis');

	async function updateApi(updated: ApiCreate) {
		let success = true;
		await apisService
			.update(api.id, { name: updated.name })
			.then((res) => {
				api = { ...api, ...res };
				toast.success(m.api_updated_successfully());
			})
			.catch((e) => {
				axiosErrorToast(e);
				success = false;
			});
		return success;
	}

	async function updatePermissions() {
		await apisService
			.updatePermissions(api.id, permissions)
			.then((res) => {
				permissions = res.permissions.map((p) => ({
					key: p.key,
					name: p.name,
					description: p.description ?? ''
				}));
				toast.success(m.api_permissions_updated_successfully());
			})
			.catch(axiosErrorToast);
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
	<div class="mt-5 flex justify-end">
		<Button usePromiseLoading onclick={updatePermissions}>{m.save()}</Button>
	</div>
</CollapsibleCard>
