// Design tokens mirrored from the frontend's app.css (Mono color theme, light) so emails match the app
// Email clients have no CSS variable or oklch support, so the values are the resolved hex equivalents
export const colors = {
	canvas: '#fafafa',
	surface: '#ffffff',
	border: '#e8e8e8',
	well: '#f5f5f5',
	fg: '#0a0a0a',
	fgMuted: '#5e5e5e',
	primary: '#0a0a0a',
	primaryForeground: '#ffffff'
};

export const fonts = {
	sans: "'DM Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif",
	display: "'Young Serif', Georgia, 'Times New Roman', serif",
	mono: "'JetBrains Mono', 'SF Mono', Menlo, Consolas, 'Liberation Mono', monospace"
};

export const radius = {
	card: '24px',
	box: '16px',
	tile: '12px',
	control: '10px'
};
