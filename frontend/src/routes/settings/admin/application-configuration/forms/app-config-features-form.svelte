<script lang="ts">
	import { invalidate } from '$app/navigation';
	import SettingRow from '#lib/components/setting-row.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import { Switch } from '#lib/components/ui/switch/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { AllAppConfig } from '#lib/types/application-configuration.type.ts';
	import type { Feature, Features } from '#lib/types/feature.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { featureSections, type FeatureRow } from '#lib/utils/feature-util.ts';
	import { toast } from 'svelte-sonner';

	let {
		features,
		callback
	}: {
		features: Features;
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

<fieldset class="flex min-w-0 flex-col gap-10" disabled={$appConfigStore.uiConfigDisabled}>
	{#each sections as section (section.title)}
		<SettingsSection title={section.title} description={section.description}>
			{#each section.rows as row (row.id)}
				<SettingRow
					title={row.title}
					description={`${row.description} ${row.adds}`}
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
		</SettingsSection>
	{/each}
</fieldset>
