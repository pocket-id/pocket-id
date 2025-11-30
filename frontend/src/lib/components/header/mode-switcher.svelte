<script lang="ts">
	import MoonIcon from '@lucide/svelte/icons/moon';
	import SunIcon from '@lucide/svelte/icons/sun';

	import { buttonVariants } from '$lib/components/ui/button/index.js';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import { m } from '$lib/paraglide/messages';
	import { mode, resetMode, setMode } from 'mode-watcher';

	const isDark = $derived(mode.current === 'dark');
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger class={buttonVariants({ variant: 'ghost', size: 'icon' })}>
		<SunIcon
			class="h-[1.2rem] w-[1.2rem] !transition-all {isDark
				? '-rotate-90 scale-0'
				: 'rotate-0 scale-100'}"
		/>
		<MoonIcon
			class="absolute h-[1.2rem] w-[1.2rem] !transition-all {isDark
				? 'rotate-0 scale-100'
				: 'rotate-90 scale-0'}"
		/>
		<span class="sr-only">Toggle theme</span>
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="end">
		<DropdownMenu.Item onclick={() => setMode('light')}>{m.light()}</DropdownMenu.Item>
		<DropdownMenu.Item onclick={() => setMode('dark')}>{m.dark()}</DropdownMenu.Item>
		<DropdownMenu.Item onclick={() => resetMode()}>{m.system()}</DropdownMenu.Item>
	</DropdownMenu.Content>
</DropdownMenu.Root>
