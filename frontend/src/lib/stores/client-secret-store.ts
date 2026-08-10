import { writable } from 'svelte/store';

// Holds the clear-text value of the client secrets created during the current page visit, keyed by secret ID.
// The server never returns those values again, so they are shown until the user navigates away and then forgotten.
const clientSecretStore = writable<Record<string, string>>({});

const set = (secretId: string, secret: string) => {
	clientSecretStore.update((secrets) => ({ ...secrets, [secretId]: secret }));
};

const remove = (secretId: string) => {
	clientSecretStore.update((secrets) => {
		const remaining = { ...secrets };
		delete remaining[secretId];
		return remaining;
	});
};

const clear = () => {
	clientSecretStore.set({});
};

export default {
	subscribe: clientSecretStore.subscribe,
	set,
	remove,
	clear
};
