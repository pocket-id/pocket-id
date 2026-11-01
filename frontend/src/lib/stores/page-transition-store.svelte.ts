// True while a settings page crossfade runs; the browser takes the pointer off the page for it, so hover events in that window are not real moves
export const pageTransition = $state({ running: false });
