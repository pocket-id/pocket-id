<script lang="ts">
	import { goto } from '$app/navigation';
	import OidcClientAvatar from '#lib/components/oidc-client-avatar.svelte';
	import * as Avatar from '#lib/components/ui/avatar/index.ts';
	import * as Command from '#lib/components/ui/command/index.ts';
	import * as Dialog from '#lib/components/ui/dialog/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import ApisService from '#lib/services/apis-service.ts';
	import OIDCService from '#lib/services/oidc-service.ts';
	import UserGroupService from '#lib/services/user-group-service.ts';
	import UserService from '#lib/services/user-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import { commandPalette } from '#lib/stores/command-palette-store.svelte.ts';
	import { cachedProfilePicture } from '#lib/utils/cached-image-util.ts';
	import { encodeClientIdParam } from '#lib/utils/client-id-util.ts';
	import { setModeInstantly } from '#lib/utils/color-theme-util.ts';
	import { signOut } from '#lib/utils/sign-out-util.ts';
	import {
		ArrowUpRight,
		LogOut,
		Monitor,
		Moon,
		Plus,
		Search,
		Server,
		Settings2,
		SlidersHorizontal,
		Sun,
		UsersRound
	} from '@lucide/svelte';
	import { Command as CommandPrimitive } from 'bits-ui';
	import { userPrefersMode } from 'mode-watcher';
	import { PersistedState } from 'runed';
	import type { Component } from 'svelte';
	import type { Features } from '#lib/types/feature.type.ts';
	import { featureSections } from '#lib/utils/feature-util.ts';
	import type { NavGroup } from './nav-items.ts';

	let {
		groups,
		features,
		isAdmin
	}: { groups: NavGroup[]; features: Features | null; isAdmin: boolean } = $props();

	type Lead =
		| { kind: 'icon'; icon: Component }
		| { kind: 'user'; id: string; initials: string }
		| { kind: 'client'; id: string; name: string; hasLogo: boolean };
	type Item = {
		key: string;
		label: string;
		context?: string;
		mono?: boolean;
		lead: Lead;
		checked?: boolean;
		external?: boolean;
		// Search-only items stay out of the list until someone types
		searchOnly?: boolean;
		run: () => void;
	};
	type Recent = {
		key: string;
		label: string;
		context?: string;
		mono?: boolean;
		lead?: Exclude<Lead, { kind: 'icon' }>;
		href: string;
	};

	const userService = new UserService();
	const groupService = new UserGroupService();
	const oidcService = new OIDCService();
	const apisService = new ApisService();

	let query = $state('');
	// The page crossfade snapshots the old page, so a palette still fading out would show under the new one
	let jumping = $state(false);
	const q = $derived(query.trim().toLowerCase());
	const recent = new PersistedState<Recent[]>('palette-recent', []);
	const icon = (i: Component): Lead => ({ kind: 'icon', icon: i });
	const matches = (item: Item) =>
		item.label.toLowerCase().includes(q) || !!item.context?.toLowerCase().includes(q);

	function jump(href: string) {
		jumping = true;
		goto(href);
	}

	// Destinations remember themselves, so the empty palette starts with the last three
	function link(
		key: string,
		label: string,
		href: string,
		lead: Lead,
		context?: string,
		mono?: boolean
	): Item {
		return {
			key,
			label,
			context,
			mono,
			lead,
			run: () => {
				recent.current = [
					{ key, label, context, mono, lead: lead.kind === 'icon' ? undefined : lead, href },
					...recent.current.filter((r) => r.key !== key)
				].slice(0, 3);
				jump(href);
			}
		};
	}

	const navigation = $derived(
		groups.flatMap((g, gi) =>
			g.items.map((i) =>
				link(i.href, i.label, i.href, icon(i.icon), gi === 0 ? undefined : g.label)
			)
		)
	);

	const configUrl = '/settings/admin/application-configuration';
	const settings = $derived.by(() => {
		if (!isAdmin) return [];
		const category = (id: string, label: string) =>
			link(`config:${id}`, label, `${configUrl}/${id}`, icon(SlidersHorizontal), m.configuration());
		const setting = (label: string, id: string, categoryLabel: string): Item => ({
			...link(`setting:${label}`, label, `${configUrl}/${id}`, icon(Settings2), categoryLabel),
			searchOnly: true
		});
		const ldap = !!features?.ldap.enabled;
		return [
			category('general', m.general()),
			category('features', m.config_tab_features()),
			category('signup', m.config_tab_signup()),
			category('passkeys', m.passkeys()),
			category('email', m.email()),
			...(ldap ? [category('ldap', m.ldap())] : []),
			category('oidc', m.oidc()),
			category('branding', m.config_tab_branding()),
			setting(m.application_name(), 'general', m.general()),
			setting(m.session_duration(), 'general', m.general()),
			setting(m.color_theme(), 'general', m.general()),
			setting(m.disable_animations(), 'general', m.general()),
			setting(m.user_verification(), 'passkeys', m.passkeys()),
			setting(m.email_verification(), 'email', m.email()),
			...(ldap ? [setting(m.config_server_url(), 'ldap', m.ldap())] : []),
			// A feature that is off is found on Features, where it can be turned on
			...featureSections($appConfigStore.appName)
				.flatMap((section) => section.rows)
				.filter((row) => !features?.[row.id].enabled)
				.map((row) => setting(row.title, 'features', m.config_tab_features())),
			setting(m.logo(), 'branding', m.config_tab_branding()),
			setting(m.config_signin_background(), 'branding', m.config_tab_branding())
		];
	});

	const actions = $derived.by(() => {
		const create = (label: string, path: string): Item => ({
			key: `new:${path}`,
			label,
			lead: icon(Plus),
			run: () => jump(`/settings/admin/${path}?new`)
		});
		const list: Item[] = isAdmin
			? [
					create(m.add_user(), 'users'),
					create(m.add_group(), 'user-groups'),
					create(m.add_oidc_client(), 'oidc-clients'),
					...(features?.claimMappings.enabled ? [create(m.add_policy(), 'claim-mappings')] : []),
					...(features?.apis.enabled ? [create(m.add_api(), 'apis')] : []),
					...(features?.apiKeys.enabled ? [create(m.create_api_key(), 'api-keys')] : [])
				]
			: [];
		list.push({
			key: 'sign-out',
			label: m.sign_out(),
			lead: icon(LogOut),
			searchOnly: true,
			run: signOut
		});
		const modes = [
			['light', m.light(), Sun],
			['dark', m.dark(), Moon],
			['system', m.system(), Monitor]
		] as const;
		for (const [mode, label, modeIcon] of modes) {
			list.push({
				key: `mode:${mode}`,
				label,
				context: m.appearance(),
				lead: icon(modeIcon),
				checked: userPrefersMode.current === mode,
				searchOnly: true,
				run: () => setModeInstantly(mode)
			});
		}
		return list;
	});

	// People and apps come from the server, so they load after the local results and never block them
	let entities = $state<{ label: string; items: Item[] }[]>([]);
	let searchStatus = $state<'idle' | 'searching' | 'done' | 'failed'>('idle');
	let searchSeq = 0;

	async function searchEntities(term: string) {
		const seq = ++searchSeq;
		const options = { search: term, pagination: { page: 1, limit: 5 } };
		const spinner = setTimeout(() => seq === searchSeq && (searchStatus = 'searching'), 150);
		try {
			const [apps, users, userGroups, clients, apis] = await Promise.all([
				oidcService.listOwnAccessibleClients(options),
				isAdmin ? userService.list(options) : null,
				isAdmin ? groupService.list(options) : null,
				isAdmin ? oidcService.listClients(options) : null,
				isAdmin ? apisService.list(options) : null
			]);
			if (seq !== searchSeq) return;
			const found = [
				{
					label: m.my_apps(),
					items: apps.data
						.filter((a) => a.launchURL)
						.map((a): Item => ({
							key: `app:${a.id}`,
							label: a.name,
							context: a.launchURL!.replace(/^https?:\/\//, ''),
							lead: { kind: 'client', id: a.id, name: a.name, hasLogo: a.hasLogo },
							external: true,
							run: () => window.open(a.launchURL, '_blank', 'noopener,noreferrer')
						}))
				},
				{
					label: m.users(),
					items: (users?.data ?? []).map((u) =>
						link(
							`user:${u.id}`,
							u.displayName,
							`/settings/admin/users/${u.id}`,
							{
								kind: 'user',
								id: u.id,
								initials: (u.firstName?.[0] ?? '') + (u.lastName?.[0] ?? '')
							},
							u.email
						)
					)
				},
				{
					label: m.groups(),
					items: (userGroups?.data ?? []).map((g) =>
						link(
							`group:${g.id}`,
							g.friendlyName,
							`/settings/admin/user-groups/${g.id}`,
							icon(UsersRound),
							g.name
						)
					)
				},
				{
					label: m.oidc_clients(),
					items: (clients?.data ?? []).map((c) =>
						link(
							`client:${c.id}`,
							c.name,
							`/settings/admin/oidc-clients/${encodeClientIdParam(c.id)}`,
							{ kind: 'client', id: c.id, name: c.name, hasLogo: c.hasLogo },
							c.id,
							true
						)
					)
				},
				{
					label: m.apis(),
					items: (apis?.data ?? []).map((a) =>
						link(
							`api:${a.id}`,
							a.name,
							`/settings/admin/apis/${a.id}`,
							icon(Server),
							a.resource,
							true
						)
					)
				}
			];
			entities = found.filter((g) => g.items.length);
			searchStatus = 'done';
		} catch {
			if (seq === searchSeq) searchStatus = 'failed';
		} finally {
			clearTimeout(spinner);
		}
	}

	$effect(() => {
		const term = q;
		entities = [];
		searchStatus = 'idle';
		// A search still running for the previous query must not fill in results for this one
		searchSeq++;
		if (!term) return;
		const timer = setTimeout(() => searchEntities(term), 200);
		return () => clearTimeout(timer);
	});

	// Icon components cannot be stored, so a recent entry's icon is found again from its key
	function recentIcon(key: string): Component {
		const kind = key.split(':')[0];
		if (kind === 'group') return UsersRound;
		if (kind === 'api') return Server;
		if (kind === 'config') return SlidersHorizontal;
		if (kind === 'setting') return Settings2;
		const nav = navigation.find((n) => n.key === key);
		return nav?.lead.kind === 'icon' ? nav.lead.icon : Search;
	}

	const sections = $derived.by(() => {
		if (!q) {
			const visible = (items: Item[]) => items.filter((i) => !i.searchOnly);
			const recentItems = recent.current.map((r) =>
				link(r.key, r.label, r.href, r.lead ?? icon(recentIcon(r.key)), r.context, r.mono)
			);
			return [
				{ label: m.palette_recent(), items: recentItems },
				{ label: m.palette_actions(), items: visible(actions) },
				{ label: m.palette_navigation(), items: navigation },
				{ label: m.palette_settings(), items: visible(settings) }
			].filter((s) => s.items.length);
		}
		return [
			{ label: m.palette_actions(), items: actions.filter(matches) },
			{ label: m.palette_navigation(), items: navigation.filter(matches) },
			{ label: m.palette_settings(), items: settings.filter(matches) },
			...entities
		].filter((s) => s.items.length);
	});

	function run(item: Item) {
		commandPalette.open = false;
		item.run();
	}

	function onKeydown(e: KeyboardEvent) {
		if (e.key.toLowerCase() === 'k' && (e.metaKey || e.ctrlKey)) {
			e.preventDefault();
			commandPalette.open = !commandPalette.open;
		}
	}

	$effect(() => {
		if (commandPalette.open) jumping = false;
		else query = '';
	});
</script>

<svelte:window onkeydown={onKeydown} />

<Dialog.Root bind:open={commandPalette.open}>
	<Dialog.Content
		showCloseButton={false}
		overlayClass={jumping ? 'data-closed:duration-0' : undefined}
		class={[
			'bg-popover/72 border-fg/6 dark:border-fg/10 [@media(prefers-reduced-transparency:reduce)]:bg-popover! top-[15%] flex h-[min(70dvh,600px)] max-w-(--palette-w) translate-y-0 flex-col gap-0 overflow-hidden rounded-lg p-0 backdrop-blur-[40px] backdrop-saturate-150 max-sm:inset-0 max-sm:h-dvh max-sm:max-h-none max-sm:w-full max-sm:max-w-none max-sm:translate-x-0 max-sm:rounded-none max-sm:border-0 max-sm:pb-[env(safe-area-inset-bottom)] max-sm:before:hidden [@media(prefers-reduced-transparency:reduce)]:backdrop-filter-none!',
			jumping && 'data-closed:duration-0'
		]}
	>
		<Dialog.Title class="sr-only">{m.command_menu()}</Dialog.Title>
		<Command.Root shouldFilter={false} loop class="p-0">
			<div
				class="flex h-12 shrink-0 items-center gap-3 px-4 shadow-[inset_0_-1px_0_var(--border)] max-sm:h-[calc(52px+env(safe-area-inset-top))] max-sm:pt-[env(safe-area-inset-top)]"
			>
				<Search class="text-fg-muted size-4 shrink-0" />
				<CommandPrimitive.Input
					bind:value={query}
					placeholder={m.search_or_jump_to()}
					class="text-body-l placeholder:text-fg-muted h-full min-w-0 flex-1 bg-transparent outline-hidden max-sm:text-[16px]"
				/>
				<Dialog.Close class="text-fg-muted hover:text-fg text-control sm:hidden"
					>{m.cancel()}</Dialog.Close
				>
			</div>
			<Command.List class="max-h-none min-h-0 flex-1 p-2">
				{#each sections as section (section.label)}
					<Command.Group heading={section.label} class="py-0">
						{#each section.items as item (item.key)}
							<Command.Item
								value={item.key}
								onSelect={() => run(item)}
								data-checked={item.checked}
								class="text-body min-h-11 gap-3 px-2 sm:min-h-10"
							>
								{#if item.lead.kind === 'user'}
									<Avatar.Root class="size-5">
										<Avatar.Image src={cachedProfilePicture.getUrl(item.lead.id)} alt="" />
										<Avatar.Fallback class="text-[9px]">{item.lead.initials}</Avatar.Fallback>
									</Avatar.Root>
								{:else if item.lead.kind === 'client'}
									<OidcClientAvatar
										id={item.lead.id}
										name={item.lead.name}
										hasLogo={item.lead.hasLogo}
										size={20}
									/>
								{:else}
									<span class="grid w-5 place-items-center"><item.lead.icon /></span>
								{/if}
								<span class="truncate">{item.label}</span>
								{#if item.context}
									<span
										class={[
											'text-fg-muted min-w-1/3 flex-1 truncate',
											item.mono && 'text-mono-s font-mono'
										]}>{item.context}</span
									>
								{/if}
								{#if item.external}<ArrowUpRight class="ml-auto size-3" />{/if}
							</Command.Item>
						{/each}
					</Command.Group>
				{/each}
				{#if searchStatus === 'searching'}
					<p class="text-control text-fg-muted px-2 py-2">{m.palette_searching()}</p>
				{:else if searchStatus === 'failed'}
					<p class="text-control text-fg-muted px-2 py-2">{m.palette_search_failed()}</p>
				{:else if searchStatus === 'done' && !sections.length}
					<p class="text-control text-fg-muted px-2 py-2">
						{m.palette_no_results({ query: query.trim() })}
					</p>
				{/if}
			</Command.List>
			<div
				class="text-caption text-fg-muted flex h-9 shrink-0 items-center gap-4 px-4 shadow-[inset_0_1px_0_var(--border)] max-sm:hidden"
			>
				<span class="flex items-center gap-1.5"
					><kbd class="keycap">↑</kbd><kbd class="keycap">↓</kbd>{m.palette_hint_navigate()}</span
				>
				<span class="flex items-center gap-1.5"
					><kbd class="keycap">↵</kbd>{m.palette_hint_open()}</span
				>
				<span class="flex items-center gap-1.5"
					><kbd class="keycap">esc</kbd>{m.palette_hint_close()}</span
				>
			</div>
		</Command.Root>
	</Dialog.Content>
</Dialog.Root>
