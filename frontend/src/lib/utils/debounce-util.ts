export function debounced<T extends (...args: any[]) => any>(
	func: T,
	delay: number,
	onLoadingChange?: (loading: boolean) => void
) {
	let debounceTimeout: ReturnType<typeof setTimeout>;
	let lastCall = 0;

	return (...args: Parameters<T>) => {
		if (debounceTimeout !== undefined) {
			clearTimeout(debounceTimeout);
		}

		const call = ++lastCall;
		onLoadingChange?.(true);

		debounceTimeout = setTimeout(async () => {
			try {
				await func(...args);
			} finally {
				// An earlier call that finishes while a newer one is pending or running must not end the loading state
				if (call === lastCall) onLoadingChange?.(false);
			}
		}, delay);
	};
}
