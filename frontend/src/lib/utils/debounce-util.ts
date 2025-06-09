export function debounced<T extends (...args: any[]) => any>(
	func: T,
	delay: number,
	onLoadingChange?: (loading: boolean) => void
) {
	let debounceTimeout: ReturnType<typeof setTimeout>;

	return (...args: Parameters<T>) => {
		if (debounceTimeout !== undefined) {
			clearTimeout(debounceTimeout);
		}

		onLoadingChange?.(true);

		debounceTimeout = setTimeout(async () => {
			try {
				await func(...args);
			} finally {
				onLoadingChange?.(false);
			}
		}, delay);
	};
}
