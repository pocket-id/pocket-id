import { onDestroy } from 'svelte';

// Returns a runner where each call cancels the one before it, so a slower older request can never overwrite the result of a newer one
// Create it while the component initialises, since destroying the component cancels the request that is still running
export function latestRequest() {
	let controller: AbortController | undefined;
	onDestroy(() => controller?.abort());

	return async (task: (signal: AbortSignal) => Promise<void>) => {
		controller?.abort();
		const current = new AbortController();
		controller = current;

		// A cancelled task has been replaced by a newer one, so whatever it throws no longer matters
		try {
			await task(current.signal);
		} catch (e) {
			if (!current.signal.aborted) throw e;
		}
	};
}
