let open = $state(false);
let waiters: ((signedIn: boolean) => void)[] = [];

// Requests that failed because the session ended wait here until the person signs in again or gives up
export const signInAgain = {
	get open() {
		return open;
	},
	wait() {
		open = true;
		return new Promise<boolean>((resolve) => waiters.push(resolve));
	},
	finish(signedIn: boolean) {
		open = false;
		waiters.forEach((resolve) => resolve(signedIn));
		waiters = [];
	}
};
