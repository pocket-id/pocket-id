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
		const accent = accentColors.find((a) => a.value === accentValue);
		if (accent) {
			// Parse the OKLCH values to create a faded version for rings
			const match = accent.cssVar.match(/oklch\(([^)]+)\)/);
			if (match) {
				const [l, c, h] = match[1].split(' ').map((v) => parseFloat(v));
				const fadedRing = `oklch(${Math.min(l + 0.15, 1)} ${c * 0.6} ${h} / 0.3)`;

				document.documentElement.style.setProperty('--primary', accent.cssVar);
				document.documentElement.style.setProperty('--primary-foreground', accent.foregroundVar);
				document.documentElement.style.setProperty('--ring', fadedRing);
				document.documentElement.style.setProperty('--sidebar-ring', fadedRing);
			}
		}
	} else {
		document.documentElement.style.removeProperty('--primary');
		document.documentElement.style.removeProperty('--primary-foreground');
		document.documentElement.style.removeProperty('--ring');
		document.documentElement.style.removeProperty('--sidebar-ring');
	}
}
