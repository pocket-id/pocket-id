export function applyAccentColor(accentValue: string) {
	if (accentValue === 'default') {
		document.documentElement.style.removeProperty('--primary');
		document.documentElement.style.removeProperty('--primary-foreground');
		document.documentElement.style.removeProperty('--ring');
		document.documentElement.style.removeProperty('--sidebar-ring');
		return;
	}

	document.documentElement.style.setProperty('--primary', accentValue);

	// Smart foreground color selection based on brightness
	const foregroundColor = getContrastingForeground(accentValue);
	document.documentElement.style.setProperty('--primary-foreground', foregroundColor);

	// Create proper ring colors based on input format
	const ringColor = `color-mix(in srgb, ${accentValue} 50%, transparent)`;
	document.documentElement.style.setProperty('--ring', ringColor);
	document.documentElement.style.setProperty('--sidebar-ring', ringColor);
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
