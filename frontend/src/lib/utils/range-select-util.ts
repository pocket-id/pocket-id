// Shift-clicking a checkbox gives every item between it and the previously clicked one the state the clicked checkbox toggles to
// Returns the new selection for a shift-click, or undefined so a plain click toggles only its own checkbox
export function createRangeSelect() {
	let anchor: string | undefined;

	return (event: MouseEvent, ids: string[], selected: string[], id: string) => {
		const from = event.shiftKey && anchor ? ids.indexOf(anchor) : -1;
		anchor = id;
		if (from === -1) return;

		// bits-ui skips its own toggle when an earlier click handler prevents the default
		event.preventDefault();
		const to = ids.indexOf(id);
		const range = ids.slice(Math.min(from, to), Math.max(from, to) + 1);
		return selected.includes(id)
			? selected.filter((selectedId) => !range.includes(selectedId))
			: Array.from(new Set([...selected, ...range]));
	};
}
