<script lang="ts">
	import UserService from '#lib/services/user-service.ts';
	import { signInAgain } from '#lib/stores/sign-in-again-store.svelte.ts';
	import userStore from '#lib/stores/user-store.ts';
	import { getAxiosErrorMessage } from '#lib/utils/error-util.ts';
	import { Button } from '@pocket-id/ui/button';

	const userService = new UserService();

	let log = $state<{ time: string; text: string; ok?: boolean }[]>([]);

	function add(text: string, ok?: boolean) {
		log.unshift({ time: new Date().toLocaleTimeString(), text, ok });
	}

	// Plain fetch ends the session on the server without going through the interceptor or clearing the user store
	async function endSession() {
		const res = await fetch('/api/webauthn/logout', { method: 'POST' });
		add(`Session ended on the server (${res.status}), the page still thinks it's signed in`);
	}

	async function request(label: string) {
		add(`${label}: sent`);
		try {
			const user = await userService.getCurrent();
			add(`${label}: succeeded as ${user.email}`, true);
		} catch (e) {
			add(`${label}: failed with "${getAxiosErrorMessage(e)}"`, false);
		}
	}

	async function openOnly() {
		const signedIn = await signInAgain.wait();
		add(`Dialog closed, signed in again: ${signedIn}`, signedIn);
	}
</script>

<svelte:head>
	<title>Sign in again preview</title>
</svelte:head>

<main class="mx-auto flex max-w-xl flex-col gap-6 px-4 py-10">
	<div>
		<h1 class="font-display text-display-m mb-2">Sign in again</h1>
		<p class="text-body text-fg-muted">
			Signed in as {$userStore?.email ?? 'nobody'}. End the session, then send a request to see the
			dialog take over and replay it.
		</p>
	</div>

	<div class="flex flex-wrap gap-2">
		<Button variant="destructive" onclick={endSession}>1. End session</Button>
		<Button onclick={() => request('Request')}>2. Send a request</Button>
		<Button
			variant="outline"
			onclick={() => ['A', 'B', 'C'].forEach((label) => request(`Request ${label}`))}
		>
			Send three at once
		</Button>
		<Button variant="outline" onclick={openOnly}>Open dialog only</Button>
	</div>

	<ol class="bg-surface flex min-h-40 flex-col gap-1 rounded-md border p-4 font-mono text-sm">
		{#each log as entry, i (log.length - i)}
			<li
				class={entry.ok === true
					? 'text-success-solid'
					: entry.ok === false
						? 'text-danger-solid'
						: ''}
			>
				<span class="text-fg-muted">{entry.time}</span>
				{entry.text}
			</li>
		{:else}
			<li class="text-fg-muted">Nothing yet</li>
		{/each}
	</ol>
</main>
