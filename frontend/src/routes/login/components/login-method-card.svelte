<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { LucideChevronRight } from '@lucide/svelte';
	import * as Card from '$lib/components/ui/card';
	import type { Component } from 'svelte';

	interface Props {
		icon: Component;
		title: string;
		description: string;
		href?: string;
		onclick?: () => void | Promise<void>;
		searchParams?: string;
	}

	let { icon: Icon, title, description, href, onclick, searchParams = '' }: Props = $props();
</script>

{#if href}
	<a href={href + searchParams}>
		<Card.Root>
			<Card.Content class="flex items-center justify-between px-4">
				<div class="flex gap-3">
					<Icon class="text-primary size-7" />
					<div class="text-start">
						<h3 class="text-lg font-semibold">{title}</h3>
						<p class="text-muted-foreground text-sm">{description}</p>
					</div>
				</div>
				<Button variant="ghost">
					<LucideChevronRight class="size-5" />
				</Button>
			</Card.Content>
		</Card.Root>
	</a>
{:else if onclick}
	<button onclick={onclick}>
		<Card.Root>
			<Card.Content class="flex items-center justify-between px-4">
				<div class="flex gap-3">
					<Icon class="text-primary size-7" />
					<div class="text-start">
						<h3 class="text-lg font-semibold">{title}</h3>
						<p class="text-muted-foreground text-sm">{description}</p>
					</div>
				</div>
				<Button variant="ghost">
					<LucideChevronRight class="size-5" />
				</Button>
			</Card.Content>
		</Card.Root>
	</button>
{/if}
