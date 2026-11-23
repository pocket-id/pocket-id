<script lang="ts">
	import { invalidate } from '$app/navigation';
	import SettingRow from '#lib/components/setting-row.svelte';
	import SettingsCard from '#lib/components/settings-card.svelte';
	import * as Alert from '@pocket-id/ui/alert';
	import { Switch } from '@pocket-id/ui/switch';
	import { m } from '#lib/paraglide/messages.js';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { AllAppConfig } from '#lib/types/application-configuration.type.ts';
	import type { Feature, Features } from '#lib/types/feature.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { featureSections, type FeatureRow } from '#lib/utils/feature-util.ts';
	import LockIcon from '~icons/solar/lock-keyhole-minimalistic-bold-duotone';
	import { toast } from 'svelte-sonner';

	let {
		features,
		envOverrides,
		callback
	}: {
		features: Features;
		envOverrides: Set<keyof AllAppConfig>;
		callback: (appConfig: Partial<AllAppConfig>) => Promise<void>;
	} = $props();

	let saving = $state<Feature | null>(null);
	const sections = $derived(featureSections($appConfigStore.appName));

	// A switch applies at once, so the page or tab it adds appears while the admin is still here
	async function toggle(row: FeatureRow, enabled: boolean) {
		const enabledFeatures = (Object.keys(features) as Feature[]).filter((id) =>
			id === row.id ? enabled : features[id].enabled
		);
		saving = row.id;
		try {
			await callback({ enabledFeatures });
			await invalidate('app:features');
			toast.success(
				enabled
					? m.feature_turned_on({ feature: row.title })
					: m.feature_turned_off({ feature: row.title })
			);
		} catch (e) {
			axiosErrorToast(e);
		} finally {
			saving = null;
		}
	}
</script>

<fieldset
	class="flex min-w-0 flex-col gap-4"
	disabled={$appConfigStore.uiConfigDisabled || envOverrides.has('enabledFeatures')}
>
	{#if envOverrides.has('enabledFeatures')}
		<Alert.Root variant="info">
			<LockIcon />
			<Alert.Title>{m.config_set_by_env()}</Alert.Title>
			<Alert.Description>{m.config_set_by_env_description()}</Alert.Description>
		</Alert.Root>
	{/if}
	{#each sections as section (section.title)}
		<SettingsCard title={section.title}>
			{#each section.rows as row (row.id)}
				<SettingRow
					title={row.title}
					description={row.description}
					consequence={features[row.id].inUse ? row.inUse : undefined}
					icon={row.icon}
				>
					{#snippet control(props)}
						<Switch
							{...props}
							bind:checked={() => features[row.id].enabled, (checked) => toggle(row, checked)}
							disabled={features[row.id].inUse || saving !== null}
							aria-busy={saving === row.id}
						/>
					{/snippet}
				</SettingRow>
			{/each}
		</SettingsCard>
	{/each}
</fieldset>
