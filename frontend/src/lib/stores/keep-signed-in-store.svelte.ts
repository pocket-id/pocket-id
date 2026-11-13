const storageKey = 'keep-signed-in';
const stored = localStorage.getItem(storageKey);

let current = $state(stored === 'true');
let decided = $state(stored !== null);
let finishAsking = $state<() => void>();

// The choice stays with the browser only when the user asks for that, so a shared computer can keep asking everyone who signs in on it
// Until a browser has a choice, its sign-ins start a session that ends when it closes, and the user is asked once signed in
export const keepSignedIn = {
	get current() {
		return current;
	},
	get asking() {
		return !!finishAsking;
	},
	// Resolves once the user has chosen, or right away when this browser already has a choice
	async ask() {
		if (decided) return;
		await new Promise<void>((resolve) => (finishAsking = resolve));
	},
	choose(keep: boolean, remember: boolean) {
		if (remember) {
			current = keep;
			decided = true;
			localStorage.setItem(storageKey, String(keep));
		}
		finishAsking?.();
		finishAsking = undefined;
	},
	// A page left while asking never continues its sign-in, which keeps the session that ends with the browser and asks again next time
	dismiss() {
		finishAsking = undefined;
	}
};
