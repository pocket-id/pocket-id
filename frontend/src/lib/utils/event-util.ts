export function preventDefault(fn: (event: Event) => void): (event: Event) => void {
	return function (this: unknown, event) {
		event.preventDefault();
		fn.call(this, event);
	};
}

// Chrome leaves a partly visible control where it is when it takes focus, so horizontal scrollers bring it into view, ring included through their scroll padding
export function revealFocused(event: FocusEvent) {
	(event.target as HTMLElement).scrollIntoView({ block: 'nearest', inline: 'nearest' });
}
