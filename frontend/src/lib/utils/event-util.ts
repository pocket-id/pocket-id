export function preventDefault(fn: (event: Event) => void): (event: Event) => void {
	return function (this: unknown, event) {
		event.preventDefault();
		fn.call(this, event);
	};
}
