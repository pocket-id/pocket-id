<script lang="ts">
	import { goto } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { m } from '$lib/paraglide/messages';
	import ApisService from '$lib/services/apis-service';
	import type { ApiCreate } from '$lib/types/api.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideMinus, LucidePlus, LucideServer } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import { slide } from 'svelte/transition';
	import ApiForm from './api-form.svelte';
	import ApiList from './api-list.svelte';

	let expandAddApi = $state(false);

	const apisService = new ApisService();

	async function createApi(api: ApiCreate) {
		let success = true;
		await apisService
			.create(api)
			.then((createdApi) => {
				toast.success(m.api_created_successfully());
				goto(`/settings/admin/apis/${createdApi.id}`);
			})
			.catch((e) => {
				axiosErrorToast(e);
				success = false;
			});
		return success;
	}
</script>

<svelte:head>
	<title>{m.apis()}</title>
</svelte:head>

<div>
	<Card.Root>
		<Card.Header>
			<div class="flex flex-wrap items-center justify-between gap-4 md:flex-nowrap">
				<div>
					<Card.Title>
						<LucidePlus class="text-primary/80 size-5" />
						{m.create_api()}
					</Card.Title>
					<Card.Description>{m.create_a_new_api_description()}</Card.Description>
				</div>
				{#if !expandAddApi}
					<Button class="w-full md:w-auto" onclick={() => (expandAddApi = true)}
						>{m.add_api()}</Button
					>
				{:else}
					<Button class="h-8 p-3" variant="ghost" onclick={() => (expandAddApi = false)}>
						<LucideMinus class="size-5" />
					</Button>
				{/if}
			</div>
		</Card.Header>
		{#if expandAddApi}
			<div transition:slide>
				<Card.Content>
					<ApiForm callback={createApi} />
				</Card.Content>
			</div>
		{/if}
	</Card.Root>
</div>

<div>
	<Card.Root>
		<Card.Header>
			<Card.Title>
				<LucideServer class="text-primary/80 size-5" />
				{m.manage_apis()}
			</Card.Title>
		</Card.Header>
		<Card.Content>
			<ApiList />
		</Card.Content>
	</Card.Root>
</div>
