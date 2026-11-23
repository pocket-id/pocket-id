export function cssVar(name: string) {
	return getComputedStyle(document.documentElement).getPropertyValue(name).trim();
}

// Durations and holds come from the motion tokens, which collapse to zero under reduced motion
export function cssMs(name: string) {
	return parseFloat(cssVar(name)) || 0;
}
