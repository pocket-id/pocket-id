<script lang="ts">
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import { m } from '$lib/paraglide/messages';
	import { setLocale } from '$lib/paraglide/runtime';
	import { Languages } from 'lucide-svelte';
	import { Button } from '../ui/button';
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger asChild let:builder>
		<Button builders={[builder]} variant="outline" size="sm" class="h-8 gap-1">
			<Languages class="h-3.5 w-3.5" />
			<span class="sr-only sm:not-sr-only sm:whitespace-nowrap"> {m.language()} </span>
		</Button>
	</DropdownMenu.Trigger>
	<DropdownMenu.Content class="min-w-40" align="start">
		<DropdownMenu.Group>
			<DropdownMenu.Label>{m.language()}</DropdownMenu.Label>
			<DropdownMenu.Item on:click={() => setLocale('en')}>{m.english()}</DropdownMenu.Item>
			<DropdownMenu.Item on:click={() => setLocale('nl')}>{m.dutch()}</DropdownMenu.Item>
		</DropdownMenu.Group>
	</DropdownMenu.Content>
</DropdownMenu.Root>
