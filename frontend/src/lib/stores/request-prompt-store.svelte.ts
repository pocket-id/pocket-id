// A dialog that failed requests wait on: they open it and learn whether the person completed it, so the request can run once more
function createRequestPrompt() {
	let open = $state(false);
	let waiters: ((completed: boolean) => void)[] = [];

	return {
		get open() {
			return open;
		},
		wait() {
			open = true;
			return new Promise<boolean>((resolve) => waiters.push(resolve));
		},
		finish(completed: boolean) {
			open = false;
			waiters.forEach((resolve) => resolve(completed));
			waiters = [];
		}
	};
}

// Requests that failed because the session ended wait here until the person signs in again or gives up
export const signInAgain = createRequestPrompt();

// Requests the server refuses outside of sudo mode wait here until the person confirms with their passkey or gives up
export const sudoPrompt = createRequestPrompt();
