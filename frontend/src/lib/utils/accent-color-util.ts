export const accentColors = [
	{ value: 'default', cssVar: 'oklch(0.205 0 0)', foregroundVar: 'oklch(0.98 0 0)' },
	{ value: 'red', cssVar: 'oklch(0.637 0.237 25.331)', foregroundVar: 'oklch(0.98 0 0)' },
	{ value: 'rose', cssVar: 'oklch(0.658 0.218 12.180)', foregroundVar: 'oklch(0.98 0 0)' },
	{ value: 'orange', cssVar: 'oklch(0.705 0.213 47.604)', foregroundVar: 'oklch(0.98 0 0)' },
	{ value: 'green', cssVar: 'oklch(0.723 0.219 149.579)', foregroundVar: 'oklch(0.98 0 0)' },
	{ value: 'blue', cssVar: 'oklch(0.623 0.214 259.815)', foregroundVar: 'oklch(0.98 0 0)' },
	{ value: 'yellow', cssVar: 'oklch(0.795 0.184 86.047)', foregroundVar: 'oklch(0.09 0 0)' },
	{ value: 'violet', cssVar: 'oklch(0.649 0.221 285.75)', foregroundVar: 'oklch(0.98 0 0)' }
];

export function applyAccentColor(accentValue: string) {
	if (accentValue !== 'default') {
		const predefinedAccent = accentColors.find((a) => a.value === accentValue);

		if (predefinedAccent) {
			// Handle predefined colors
			const match = predefinedAccent.cssVar.match(/oklch\(([^)]+)\)/);
			if (match) {
				const [l, c, h] = match[1].split(' ').map((v) => parseFloat(v));
				const fadedRing = `oklch(${Math.min(l + 0.15, 1)} ${c * 0.6} ${h} / 0.5)`;

				document.documentElement.style.setProperty('--primary', predefinedAccent.cssVar);
				document.documentElement.style.setProperty(
					'--primary-foreground',
					predefinedAccent.foregroundVar
				);
				document.documentElement.style.setProperty('--ring', fadedRing);
				document.documentElement.style.setProperty('--sidebar-ring', fadedRing);
			}
		} else {
			// Handle custom colors
			document.documentElement.style.setProperty('--primary', accentValue);

			// Smart foreground color selection based on brightness
			const foregroundColor = getContrastingForeground(accentValue);
			document.documentElement.style.setProperty('--primary-foreground', foregroundColor);

			// Create proper ring colors based on input format
			let ringColor;
			if (accentValue.startsWith('#')) {
				const hex = accentValue.slice(1);
				const r = parseInt(hex.slice(0, 2), 16);
				const g = parseInt(hex.slice(2, 4), 16);
				const b = parseInt(hex.slice(4, 6), 16);
				ringColor = `rgb(${r} ${g} ${b} / 0.5)`;
			} else if (accentValue.startsWith('hsl')) {
				ringColor = accentValue.replace('hsl(', 'hsl(').replace(')', ' / 0.5)');
			} else if (accentValue.startsWith('oklch')) {
				ringColor = accentValue.replace(')', ' / 0.5)');
			} else {
				ringColor = `color-mix(in srgb, ${accentValue} 50%, transparent)`;
			}

			document.documentElement.style.setProperty('--ring', ringColor);
			document.documentElement.style.setProperty('--sidebar-ring', ringColor);
		}
	} else {
		document.documentElement.style.removeProperty('--primary');
		document.documentElement.style.removeProperty('--primary-foreground');
		document.documentElement.style.removeProperty('--ring');
		document.documentElement.style.removeProperty('--sidebar-ring');
	}
}

function getContrastingForeground(color: string): string {
	const brightness = getColorBrightness(color);

	// Use white text for dark colors, black text for light colors
	return brightness < 0.55 ? 'oklch(0.98 0 0)' : 'oklch(0.09 0 0)';
}

function getColorBrightness(color: string): number {
	// Create a temporary element to get computed color
	const tempElement = document.createElement('div');
	tempElement.style.color = color;
	document.body.appendChild(tempElement);

	const computedColor = window.getComputedStyle(tempElement).color;
	document.body.removeChild(tempElement);

	// Parse RGB values from computed color
	const rgbMatch = computedColor.match(/rgb\((\d+),\s*(\d+),\s*(\d+)\)/);
	if (!rgbMatch) {
		// Fallback: assume medium brightness
		return 0.5;
	}

	const [, r, g, b] = rgbMatch.map(Number);

	// Calculate relative luminance using the standard formula
	// https://www.w3.org/WAI/WCAG21/Understanding/contrast-minimum.html
	const sR = r / 255;
	const sG = g / 255;
	const sB = b / 255;

	const rLinear = sR <= 0.03928 ? sR / 12.92 : Math.pow((sR + 0.055) / 1.055, 2.4);
	const gLinear = sG <= 0.03928 ? sG / 12.92 : Math.pow((sG + 0.055) / 1.055, 2.4);
	const bLinear = sB <= 0.03928 ? sB / 12.92 : Math.pow((sB + 0.055) / 1.055, 2.4);

	return 0.2126 * rLinear + 0.7152 * gLinear + 0.0722 * bLinear;
}
