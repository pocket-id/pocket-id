import { setMode } from 'mode-watcher';
import { colorThemes } from './color-themes.ts';

// A stored value is a theme id, or custom:<css color> for a custom theme built from one color, with a :dark suffix when its sidebar is dark

type Rgb = [number, number, number];
type Oklch = { l: number; c: number; h: number };
type Mode = 'light' | 'dark';

export type AccentInputs = {
	name: Mode;
	surface: Rgb;
	canvas: Rgb;
	fg: Rgb;
	inkFill: Rgb;
	inkFillFg: Rgb;
	inkHover: Rgb;
	borderControl: Rgb;
	controlBgs: Rgb[];
	semanticHues: Record<string, number>;
};

export type ColorTheme = {
	id: string;
	accent: string;
	tokens: Record<Mode, Record<string, string>>;
	accentInputs: Record<Mode, AccentInputs>;
};

export const DEFAULT_COLOR_THEME = 'mono';

// sRGB transfer functions
const toLinear = (c: number) => (c <= 0.04045 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4));
const toGamma = (c: number) => (c <= 0.0031308 ? 12.92 * c : 1.055 * Math.pow(c, 1 / 2.4) - 0.055);

// OKLCH to linear sRGB (Björn Ottosson's matrices)
function oklchToLinear({ l, c, h }: Oklch): Rgb {
	const hr = (h * Math.PI) / 180;
	const a = c * Math.cos(hr);
	const b = c * Math.sin(hr);
	const L = (l + 0.3963377774 * a + 0.2158037573 * b) ** 3;
	const M = (l - 0.1055613458 * a - 0.0638541728 * b) ** 3;
	const S = (l - 0.0894841775 * a - 1.291485548 * b) ** 3;
	return [
		4.0767416621 * L - 3.3077115913 * M + 0.2309699292 * S,
		-1.2684380046 * L + 2.6097574011 * M - 0.3413193965 * S,
		-0.0041960863 * L - 0.7034186147 * M + 1.707614701 * S
	];
}

function linearToOklch([r, g, b]: Rgb): Oklch {
	const l = Math.cbrt(0.4122214708 * r + 0.5363325363 * g + 0.0514459929 * b);
	const m = Math.cbrt(0.2119034982 * r + 0.6806995451 * g + 0.1073969566 * b);
	const s = Math.cbrt(0.0883024619 * r + 0.2817188376 * g + 0.6299787005 * b);
	const A = 1.9779984951 * l - 2.428592205 * m + 0.4505937099 * s;
	const B = 0.0259040371 * l + 0.7827717662 * m - 0.808675766 * s;
	let h = (Math.atan2(B, A) * 180) / Math.PI;
	if (h < 0) h += 360;
	return {
		l: 0.2104542553 * l + 0.793617785 * m - 0.0040720468 * s,
		c: Math.sqrt(A * A + B * B),
		h
	};
}

const inGamut = (lin: Rgb) => lin.every((v) => v >= -0.0005 && v <= 1.0005);

// Reduce chroma until the color fits sRGB, keeping lightness and hue (CSS Color 4 style)
function clampChroma(o: Oklch): Oklch {
	if (inGamut(oklchToLinear(o))) return o;
	let lo = 0;
	let hi = o.c;
	for (let i = 0; i < 24; i++) {
		const mid = (lo + hi) / 2;
		if (inGamut(oklchToLinear({ ...o, c: mid }))) lo = mid;
		else hi = mid;
	}
	return { ...o, c: lo };
}

function oklchToRgb(o: Oklch): Rgb {
	return oklchToLinear(clampChroma(o)).map((v) =>
		Math.round(Math.min(1, Math.max(0, toGamma(Math.min(1, Math.max(0, v))))) * 255)
	) as Rgb;
}

const rgbToOklch = (rgb: Rgb) => linearToOklch(rgb.map((v) => toLinear(v / 255)) as Rgb);
const hex = (rgb: Rgb) => '#' + rgb.map((v) => v.toString(16).padStart(2, '0')).join('');

function hslToRgb(h: number, s: number, l: number): Rgb {
	h = (((h % 360) + 360) % 360) / 360;
	const f = (n: number) => {
		const k = (n + h * 12) % 12;
		const a = s * Math.min(l, 1 - l);
		return l - a * Math.max(-1, Math.min(k - 3, 9 - k, 1));
	};
	return [f(0), f(8), f(4)].map((v) => Math.round(Math.min(1, Math.max(0, v)) * 255)) as Rgb;
}

// Parses #rgb, #rrggbb(aa), rgb(), hsl() and oklch(), the formats the custom color dialog and the stored presets use
export function parseColor(input: string): { rgb: Rgb; alpha: number } | null {
	const s = String(input).trim().toLowerCase();
	const alphaOf = (v: string | undefined, pct: string | undefined) =>
		v === undefined || v === '' ? 1 : pct ? +v / 100 : +v;
	let m;
	if ((m = s.match(/^#([0-9a-f]{3})([0-9a-f])?$/)))
		return {
			rgb: m[1].split('').map((x) => parseInt(x + x, 16)) as Rgb,
			alpha: m[2] ? parseInt(m[2] + m[2], 16) / 255 : 1
		};
	if ((m = s.match(/^#([0-9a-f]{6})([0-9a-f]{2})?$/))) {
		const digits = m[1];
		return {
			rgb: [0, 2, 4].map((i) => parseInt(digits.slice(i, i + 2), 16)) as Rgb,
			alpha: m[2] ? parseInt(m[2], 16) / 255 : 1
		};
	}
	if (
		(m = s.match(
			/^rgba?\(\s*([\d.]+)[\s,]+([\d.]+)[\s,]+([\d.]+)\s*(?:[,/]\s*([\d.]+)(%?))?\s*\)$/
		))
	)
		return {
			rgb: [m[1], m[2], m[3]].map((x) => Math.round(+x)) as Rgb,
			alpha: alphaOf(m[4], m[5])
		};
	if (
		(m = s.match(
			/^hsla?\(\s*([\d.]+)(?:deg)?[\s,]+([\d.]+)%[\s,]+([\d.]+)%\s*(?:[,/]\s*([\d.]+)(%?))?\s*\)$/
		))
	)
		return { rgb: hslToRgb(+m[1], +m[2] / 100, +m[3] / 100), alpha: alphaOf(m[4], m[5]) };
	if (
		(m = s.match(/^oklch\(\s*([\d.]+)(%?)\s+([\d.]+)\s+([\d.]+)\s*(?:\/\s*([\d.]+)(%?))?\s*\)$/))
	) {
		const l = m[2] ? +m[1] / 100 : +m[1];
		return { rgb: oklchToRgb({ l, c: +m[3], h: +m[4] }), alpha: alphaOf(m[5], m[6]) };
	}
	return null;
}

function luminance(rgb: Rgb) {
	const [r, g, b] = rgb.map((v) => toLinear(v / 255));
	return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function contrast(a: Rgb, b: Rgb) {
	const la = luminance(a);
	const lb = luminance(b);
	return (Math.max(la, lb) + 0.05) / (Math.min(la, lb) + 0.05);
}

const WHITE: Rgb = [255, 255, 255];
const INK_ON_ACCENT = oklchToRgb({ l: 0.17, c: 0.004, h: 75 });

// Lightness search bounds for the accent fit
const FIT_MAX_DEEPEN = 0.12;
const DARK_MIN_SURFACE_CONTRAST = 2.2;
const DARK_MAX_LIFT = 0.3;
const HOVER_STEP = 0.045;
// The near-white ink fill of dark mode needs a larger step than operator colors before its hover is visible
const DARK_INK_HOVER_STEP = 0.07;
// Controls need 3:1, and 3.2 leaves margin for rounding and antialiasing
const CONTROL_TARGET = 3.2;
const CONTROL_MAX_LIFT = 0.12;
// A fill below this contrast against the surface or canvas gets a 1px border so the button keeps its shape
const EDGE_MIN = 1.5;
// An accent within this hue window of a semantic color, at visible chroma, may read as a status
const SEMANTIC_HUE_WINDOW = 20;
const SEMANTIC_MIN_CHROMA = 0.08;

export type DerivedAccent = {
	fill: Rgb;
	fillFg: Rgb;
	hover: Rgb;
	control: Rgb;
	controlFg: Rgb;
	edge: Rgb | null;
	// The fill was darkened so white text passes, or lightened so it stays visible on the dark surface
	deepened: boolean;
	lifted: boolean;
	// The semantic color the accent sits next to, so the picker can warn that a primary may read as a status
	nearSemantic: string | null;
};

// Derives every primary token for one theme and mode from an operator color, keeping WCAG AA for text and 3:1 for controls
export function deriveAccent(input: string, theme: AccentInputs): DerivedAccent {
	const isDefault = !input || input.trim().toLowerCase() === 'default';
	const parsed = isDefault ? null : parseColor(input);
	let rgb = parsed && parsed.alpha >= 1 ? parsed.rgb : null;

	// Default, and anything unusable, uses the ink pair: near-black in light, near-white in dark
	let fill = theme.inkFill;
	let fillFg = theme.inkFillFg;
	let hover = theme.inkHover;
	let fromOperator = false;
	let deepened = false;
	let lifted = false;

	if (rgb) {
		let o = rgbToOklch(rgb);

		// Keep the operator hue, but deepen by at most FIT_MAX_DEEPEN when that lets white text pass AA
		if (contrast(WHITE, rgb) < 4.5) {
			for (let d = 0.005; d <= FIT_MAX_DEEPEN + 1e-9; d += 0.005) {
				const cand = oklchToRgb({ ...o, l: o.l - d });
				if (contrast(WHITE, cand) >= 4.5) {
					rgb = cand;
					o = rgbToOklch(cand);
					deepened = true;
					break;
				}
			}
		}

		// In dark mode, lift fills that would disappear into the surface
		if (theme.name === 'dark' && contrast(rgb, theme.surface) < DARK_MIN_SURFACE_CONTRAST) {
			for (let d = 0.01; d <= DARK_MAX_LIFT + 1e-9; d += 0.01) {
				const cand = oklchToRgb({ ...o, l: o.l + d });
				if (contrast(cand, theme.surface) >= DARK_MIN_SURFACE_CONTRAST) {
					rgb = cand;
					o = rgbToOklch(cand);
					lifted = true;
					break;
				}
			}
		}

		// Pick the text color, preferring white, and fall back to ink when neither passes
		const textColor =
			contrast(WHITE, rgb) >= 4.5
				? WHITE
				: contrast(INK_ON_ACCENT, rgb) >= 4.5
					? INK_ON_ACCENT
					: null;
		if (textColor) {
			fill = rgb;
			fillFg = textColor;
			fromOperator = true;

			// Hover moves lightness away from the text color so contrast only grows
			const dir = textColor === WHITE ? -1 : 1;
			let hl = o.l + dir * HOVER_STEP;
			if (hl < 0.1 || hl > 0.96) hl = o.l - dir * HOVER_STEP;
			const hov = oklchToRgb({ ...o, l: hl });
			hover = contrast(textColor, hov) >= 4.5 ? hov : rgb;
		}
	}

	// Switch tracks, checkboxes, radios and the focus ring need 3:1 against every background a control can sit on
	const minVs = (c: Rgb) => Math.min(...theme.controlBgs.map((b) => contrast(c, b)));
	// The glyph on a control needs 3:1 on the control; prefer the fill's own text color
	const glyphFor = (c: Rgb) =>
		[fillFg, WHITE, INK_ON_ACCENT].find((g) => contrast(g, c) >= 3) || null;
	let control: Rgb | null = null;
	if (minVs(fill) >= CONTROL_TARGET && glyphFor(fill)) control = fill;
	else if (theme.name === 'dark' && fromOperator) {
		// In dark mode, lift the control on its own before giving up, so a mid-tone brand color stays the brand color on switches and rings
		const oc = rgbToOklch(fill);
		for (let d = 0.005; d <= CONTROL_MAX_LIFT + 1e-9; d += 0.005) {
			const cand = oklchToRgb({ ...oc, l: oc.l + d });
			if (minVs(cand) >= CONTROL_TARGET && glyphFor(cand)) {
				control = cand;
				break;
			}
		}
	}

	const needsEdge =
		contrast(fill, theme.surface) < EDGE_MIN || contrast(fill, theme.canvas) < EDGE_MIN;

	let nearSemantic: string | null = null;
	if (fromOperator) {
		const fo = rgbToOklch(fill);
		if (fo.c >= SEMANTIC_MIN_CHROMA) {
			let best = SEMANTIC_HUE_WINDOW + 1e-9;
			for (const [tone, h] of Object.entries(theme.semanticHues)) {
				const dh = Math.min(Math.abs(fo.h - h), 360 - Math.abs(fo.h - h));
				if (dh <= best) {
					best = dh;
					nearSemantic = tone;
				}
			}
		}
	}

	return {
		fill,
		fillFg,
		hover,
		control: control || theme.fg,
		controlFg: control ? glyphFor(control)! : theme.inkFillFg,
		edge: needsEdge ? theme.borderControl : null,
		deepened,
		lifted,
		nearSemantic
	};
}

export function primaryVars(d: DerivedAccent): Record<string, string> {
	const rgba = (rgb: Rgb, a: number) => `rgb(${rgb.join(' ')} / ${a})`;
	return {
		primary: hex(d.fill),
		'primary-foreground': hex(d.fillFg),
		'primary-hover': hex(d.hover),
		'primary-control': hex(d.control),
		'primary-control-foreground': hex(d.controlFg),
		ring: hex(d.control),
		'ring-halo': rgba(d.control, 0.24),
		'primary-edge': d.edge ? hex(d.edge) : 'transparent'
	};
}

export function parseColorTheme(value: string): {
	id: string;
	color?: string;
	darkSidebar?: boolean;
} {
	if (!value.startsWith('custom:')) return { id: value };
	const color = value.slice('custom:'.length);
	if (color.endsWith(':dark'))
		return { id: 'custom', color: color.slice(0, -':dark'.length), darkSidebar: true };
	return { id: 'custom', color, darkSidebar: false };
}

export function customColorTheme(color: string, darkSidebar: boolean) {
	return `custom:${color}${darkSidebar ? ':dark' : ''}`;
}

const themeById = (id: string) => colorThemes.find((t) => t.id === id);

// A custom theme tints Sage's neutrals with the operator's hue at the same lightness, so contrast stays where Sage has it
// A muted color tints less and a gray one gives plain grays; the accent is the color itself
// A dark sidebar tints Midnight's sidebar tokens the same way, so the rail keeps Midnight's contrast
function customTheme(color: string, darkSidebar: boolean): ColorTheme {
	const sage = themeById('sage')!;
	const parsed = parseColor(color);
	if (!parsed || parsed.alpha < 1)
		return { ...themeById(DEFAULT_COLOR_THEME)!, id: 'custom', accent: color };
	const o = rgbToOklch(parsed.rgb);
	const k = Math.min(1, o.c / 0.12);
	const tint = (value: string) => {
		const t = rgbToOklch(parseColor(value)!.rgb);
		return hex(oklchToRgb({ l: t.l, c: t.c * k, h: o.h }));
	};
	const tokens = {} as ColorTheme['tokens'];
	const accentInputs = {} as ColorTheme['accentInputs'];
	for (const mode of ['light', 'dark'] as const) {
		tokens[mode] = Object.fromEntries(
			Object.entries(sage.tokens[mode]).map(([key, value]) => [key, tint(value)])
		);
		if (darkSidebar) {
			for (const [key, value] of Object.entries(themeById('midnight')!.tokens[mode])) {
				if (key === 'sidebar' || key.startsWith('sidebar-')) tokens[mode][key] = tint(value);
			}
		}
		const t = tokens[mode];
		const rgb = (key: string) => parseColor(t[key])!.rgb;
		accentInputs[mode] = {
			...sage.accentInputs[mode],
			surface: rgb('surface'),
			canvas: rgb('canvas'),
			fg: rgb('fg'),
			inkFill: rgb('fg'),
			inkFillFg: mode === 'light' ? WHITE : rgb('canvas'),
			inkHover:
				mode === 'light'
					? rgb('inverse-hover')
					: oklchToRgb({
							...rgbToOklch(rgb('fg')),
							l: rgbToOklch(rgb('fg')).l - DARK_INK_HOVER_STEP
						}),
			borderControl: rgb('border-strong'),
			controlBgs: [
				'canvas',
				'surface',
				'surface-raised',
				'surface-sunken',
				'fill-muted',
				'fill-hover',
				'fill-row-hover',
				'fill-selected'
			].map(rgb)
		};
	}
	return {
		...sage,
		id: 'custom',
		accent: color,
		tokens,
		accentInputs,
		sidebarVars: darkSidebar ? themeById('midnight')!.sidebarVars : undefined
	};
}

export function colorThemeFor(value: string): ColorTheme {
	const { id, color, darkSidebar } = parseColorTheme(value || DEFAULT_COLOR_THEME);
	if (id === 'custom' && color) return customTheme(color, !!darkSidebar);
	return themeById(id) || themeById(DEFAULT_COLOR_THEME)!;
}

// Writes the theme's neutrals and derived primary family for light and dark as one stylesheet that overrides app.css
export function applyColorTheme(value: string) {
	const theme = colorThemeFor(value);
	const block = (vars: Record<string, string>) =>
		Object.entries(vars)
			.map(([k, v]) => `--${k}: ${v};`)
			.join(' ');
	const css = (['light', 'dark'] as const)
		.map((mode) => {
			const vars = {
				...theme.tokens[mode],
				...primaryVars(deriveAccent(theme.accent, theme.accentInputs[mode]))
			};
			return `${mode === 'light' ? 'html:root' : 'html.dark'} { ${block(vars)} }`;
		})
		.join('\n');

	let el = document.getElementById('color-theme');
	if (!el) {
		el = document.createElement('style');
		el.id = 'color-theme';
		document.head.appendChild(el);
	}
	el.textContent = css;
}

// The light and dark halves of a theme tile: sidebar, page, selected line, text line and the fitted primary
export function colorThemePreview(value: string) {
	const theme = colorThemeFor(value);
	return (['light', 'dark'] as const).map((mode) => {
		const t = theme.tokens[mode];
		return {
			mode,
			sidebar: t.sidebar,
			page: t.canvas,
			line: t['fill-selected'],
			text: t['fg-muted'],
			primary: hex(deriveAccent(theme.accent, theme.accentInputs[mode]).fill)
		};
	});
}

// mode-watcher's own disableTransitions stays off because it also blocks transitions briefly at load, which cut the authorize animation short
export function setModeInstantly(mode: 'light' | 'dark' | 'system') {
	const root = document.documentElement;
	root.classList.add('theme-switching');
	setMode(mode);
	requestAnimationFrame(() =>
		requestAnimationFrame(() => root.classList.remove('theme-switching'))
	);
}
