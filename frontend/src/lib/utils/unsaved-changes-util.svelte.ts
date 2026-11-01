import { m } from '#lib/paraglide/messages.js';
import unsavedChanges, { type UnsavedSection } from '#lib/stores/unsaved-changes-store.svelte.ts';
import { deepCopy, deepEqual } from '#lib/utils/form-util.ts';
import { untrack } from 'svelte';
import { fromStore, type Readable } from 'svelte/store';

/**
 * Registers a section of a page (e.g. a card) with the unsaved-changes bar.
 * The section is unregistered when the calling component is destroyed.
 */
export function trackUnsavedSection(
	changes: () => number,
	save: () => Promise<void>,
	discard: () => void,
	options: { validate?: () => boolean } = {}
) {
	const changeCount = $derived.by(changes);
	const section: UnsavedSection = {
		get changes() {
			return changeCount;
		},
		validate: options.validate,
		save,
		discard
	};

	$effect(() => {
		unsavedChanges.register(section);
		return () => unsavedChanges.unregister(section);
	});
}

interface TrackedForm<T> {
	inputs: Readable<unknown>;
	changedCount: () => number;
	validate: () => T | null;
	commit: (data: T) => void;
	reset: () => void;
	clearErrors: () => void;
}

/**
 * Convenience wrapper around `trackUnsavedSection` for forms built with `createForm`.
 */
export function trackFormChanges<T>(
	getForm: () => TrackedForm<T>,
	onSave: (data: T) => Promise<unknown>,
	extra: { dirty?: () => boolean; discard?: () => void; enabled?: () => boolean } = {}
) {
	const inputs = $derived(fromStore(getForm().inputs));
	const changes = $derived.by(() => {
		// The inputs are deeply reactive, so edits are tracked through the reads in `changedCount()`
		// Reading the store on top of that picks up `commit()`, which moves the baseline the inputs are compared against without touching the inputs themselves
		void inputs.current;
		if (extra.enabled && !extra.enabled()) return 0;
		return getForm().changedCount() + (extra.dirty?.() ? 1 : 0);
	});

	// Validation errors are only meaningful while there is something to save: a field that is
	// edited back to its saved value would otherwise keep showing the error of a failed save.
	$effect(() => {
		if (changes === 0) untrack(() => getForm().clearErrors());
	});

	trackUnsavedSection(
		() => changes,
		async () => {
			const form = getForm();
			const data = form.validate();
			// The bar validates every section before saving, so this only happens if the inputs
			// changed in between.
			if (!data) throw new Error(m.could_not_save_check_the_highlighted_fields());
			await onSave(data);
			form.commit(data);
		},
		() => {
			getForm().reset();
			extra.discard?.();
		},
		{ validate: () => getForm().validate() !== null }
	);
}

/**
 * Convenience wrapper around `trackUnsavedSection` for a value that is edited in place rather
 * than through `createForm`, typically a list bound to a selection component.
 */
export function trackUnsavedValue<T>(
	read: () => T,
	write: (value: T) => void,
	save: (value: T) => Promise<unknown>
) {
	let saved = $state.raw(deepCopy(read()));

	// Reads again rather than reusing the value that was saved, so that a caller that replaces
	// the value with the server's response is taken into account.
	const markSaved = () => {
		saved = deepCopy(read());
	};

	trackUnsavedSection(
		() => {
			const value = read();
			// An object of settings counts one change per setting that differs
			if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
				const savedSettings = saved as Record<string, unknown>;
				return Object.entries(value).filter(([key, item]) => !deepEqual(item, savedSettings[key]))
					.length;
			}
			return deepEqual(value, saved) ? 0 : 1;
		},
		async () => {
			await save(deepCopy(read()));
			markSaved();
		},
		() => write(deepCopy(saved))
	);

	return { markSaved };
}
