// Design tokens mirrored from the frontend's app.css (light theme) so emails match the app
// Email clients have no CSS variable or oklch support, so the values are the resolved hex equivalents
export const colors = {
	background: '#fafafa',
	card: '#ffffff',
	border: '#e8e8e8',
	muted: '#f5f5f5',
	foreground: '#0a0a0a',
	text: '#404040',
	mutedForeground: '#5e5e5e',
	primary: '#0a0a0a',
	primaryForeground: '#ffffff'
};

export const fonts = {
	sans: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif",
	display:
		"'Funnel Display', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif",
	mono: "'SF Mono', Menlo, Consolas, 'Liberation Mono', monospace"
};

export const radius = {
	card: '24px',
	box: '16px',
	pill: '9999px'
};
