import type { User } from '$lib/types/user.type';
import { setLocale } from '$lib/utils/locale.util';
import { applyAccentColor } from '$lib/utils/accent-color-util';
import { writable } from 'svelte/store';

const userStore = writable<User | null>(null);

const setUser = (user: User) => {
	if (user.locale) {
		setLocale(user.locale, false);
	}
	if (user.accentColor) {
		applyAccentColor(user.accentColor);
	}
	userStore.set(user);
};

const clearUser = () => {
	userStore.set(null);
};

export default {
	subscribe: userStore.subscribe,
	setUser,
	clearUser
};
