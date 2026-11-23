import { getContext, setContext } from 'svelte';

// Functions rather than strings, so a locale switch re-renders without setting the labels again
export type UiLabels = {
	close: () => string;
	optional: () => string;
};

const key = Symbol('ui-labels');

const englishLabels: UiLabels = {
	close: () => 'Close',
	optional: () => 'Optional'
};

// Call during component initialization, usually in the root layout, to translate the labels the components render themselves
export function setUiLabels(labels: UiLabels) {
	setContext(key, labels);
}

export function getUiLabels(): UiLabels {
	return getContext<UiLabels>(key) ?? englishLabels;
}
