import { m } from '#lib/paraglide/messages.js';
import AppConfigService from '#lib/services/app-config-service.ts';
import type { AllAppConfig } from '#lib/types/application-configuration.type.ts';
import {
	cachedApplicationLogo,
	cachedBackgroundImage,
	cachedDefaultProfilePicture,
	cachedEmailLogo,
	cachedTitleFont
} from '#lib/utils/cached-asset-util.ts';
import { deepEqual } from '#lib/utils/form-util.ts';
import appConfigStore from '#lib/stores/application-configuration-store.ts';
import { deriveAccent, parseColor, primaryVars } from '#lib/utils/color-theme-util.ts';
import { monoTheme } from '#lib/utils/mono-theme.ts';
import { trackUnsavedSection } from '#lib/utils/unsaved-changes-util.svelte.ts';
import { mode } from 'mode-watcher';
import { fromStore } from 'svelte/store';

export type Mode = 'light' | 'dark';
export type Layout = 'classic' | 'twin-cards' | 'backdrop';
export type SurfaceRole = 'background' | 'sidebar' | 'surface';
export type Surfaces = Record<SurfaceRole, string>;
export type Font = { kind: 'serif' } | { kind: 'custom'; name: string };

export type Appearance = {
	layout: Layout;
	surfaces: Record<Mode, Surfaces>;
	// One color for both modes, which each mode adjusts until its text passes contrast, null keeps the theme's monochrome ink
	primary: string | null;
	// The font of page and dialog titles, Young Serif unless an admin uploads their own, body text always stays DM Sans
	font: Font;
	// The body text size in px, every other text size grows or shrinks with it
	textSize: number;
	// Multiplies every radius token
	radius: number;
};

export const defaultAppearance: Appearance = {
	layout: 'twin-cards',
	surfaces: {
		light: { background: '#f0f0f0', sidebar: '#fafafa', surface: '#ffffff' },
		dark: { background: '#1a1a1a', sidebar: '#161616', surface: '#111111' }
	},
	primary: null,
	font: { kind: 'serif' },
	textSize: 16,
	radius: 1.25
};

export const layouts: { value: Layout; label: string }[] = [
	{ value: 'classic', label: 'Classic' },
	{ value: 'twin-cards', label: 'Twin Cards' },
	{ value: 'backdrop', label: 'Backdrop' }
];

export const surfaceRoles: { value: SurfaceRole; label: string }[] = [
	{ value: 'background', label: 'Background' },
	{ value: 'sidebar', label: 'Sidebar' },
	{ value: 'surface', label: 'Surface' }
];

export const swatches: Record<Mode, Record<SurfaceRole, string[]>> = {
	light: {
		background: ['#ffffff', '#f5f5f5', '#f0f0f0', '#ebebeb', '#dedede'],
		sidebar: ['#ffffff', '#fafafa', '#f5f5f5', '#f0f0f0', '#e5e5e5'],
		surface: ['#ffffff', '#fcfcfc', '#fafafa', '#f7f7f7', '#f2f2f2']
	},
	dark: {
		background: ['#000000', '#0f0f0f', '#141414', '#1a1a1a', '#262626'],
		sidebar: ['#000000', '#111111', '#161616', '#1a1a1a', '#262626'],
		surface: ['#000000', '#0a0a0a', '#111111', '#1a1a1a', '#242424']
	}
};

// Teal, blue, violet and berry at about the same lightness, each at least 30 degrees of hue away from the error, warning and success colors
export const primaryColors = ['#0f766e', '#2563eb', '#7c3aed', '#b5338a'];

export const fonts: { value: Font['kind']; label: string; family: string }[] = [
	{ value: 'serif', label: 'Young Serif', family: "'Young Serif', serif" },
	// Specimens on the page show the draft, so the custom option names the draft's font
	{ value: 'custom', label: 'Custom', family: "'Appearance Draft', 'Young Serif', serif" }
];

export const textSizes: { value: number; label: string }[] = [
	{ value: 15, label: 'Small' },
	{ value: 16, label: 'Default' },
	{ value: 17, label: 'Large' }
];

export const radii: { value: number; label: string }[] = [
	{ value: 0, label: 'Square' },
	{ value: 0.5, label: 'Subtle' },
	{ value: 1, label: 'Medium' },
	{ value: 1.25, label: 'Round' },
	{ value: 1.5, label: 'Rounder' }
];

export type BrandImageKey =
	| 'logoLight'
	| 'logoDark'
	| 'background'
	| 'emailLogo'
	| 'favicon'
	| 'defaultProfilePicture';

export const brandImages: {
	key: BrandImageKey;
	title: () => string;
	name: () => string;
	label?: () => string;
	spec: () => string;
	url: () => string;
	accept: string;
	variant?: 'swatch' | 'background' | 'avatar';
	well?: 'light' | 'dark' | 'surface';
	canReset?: boolean;
	canRemove?: boolean;
	minWidth?: number;
}[] = [
	{
		key: 'logoLight',
		title: m.logo,
		name: m.config_image_light_logo,
		label: m.light,
		spec: m.config_logo_spec,
		url: () => cachedApplicationLogo.getUrl(true),
		accept: 'image/svg+xml, image/png, image/webp',
		canReset: true,
		minWidth: 256
	},
	{
		key: 'logoDark',
		title: m.logo,
		name: m.config_image_dark_logo,
		label: m.dark,
		spec: m.config_logo_spec,
		url: () => cachedApplicationLogo.getUrl(false),
		accept: 'image/svg+xml, image/png, image/webp',
		well: 'dark',
		canReset: true,
		minWidth: 256
	},
	{
		key: 'background',
		title: m.config_signin_background,
		name: m.config_image_background,
		spec: m.config_signin_background_spec,
		url: () => cachedBackgroundImage.getUrl(),
		accept: 'image/jpeg, image/png, image/webp',
		variant: 'background',
		well: 'surface',
		canRemove: true,
		minWidth: 1920
	},
	{
		key: 'emailLogo',
		title: m.email_logo,
		name: m.config_image_email_logo,
		spec: m.config_email_logo_spec,
		url: () => cachedEmailLogo.getUrl(),
		accept: 'image/png, image/jpeg'
	},
	{
		key: 'favicon',
		title: m.favicon,
		name: m.config_image_favicon,
		spec: m.config_favicon_spec,
		url: () => '/api/application-assets/favicon',
		accept: 'image/svg+xml, image/png, image/x-icon'
	},
	{
		key: 'defaultProfilePicture',
		title: m.default_profile_picture,
		name: m.config_image_default_profile_picture,
		spec: m.config_default_profile_picture_spec,
		url: () => cachedDefaultProfilePicture.getUrl(),
		accept: 'image/png, image/jpeg, image/webp',
		variant: 'avatar',
		canRemove: true
	}
];

// Older settings stored null for the theme's own color, which the defaults now spell out
const setColors = (colors?: Partial<Record<SurfaceRole, string | null>>) =>
	Object.fromEntries(Object.entries(colors ?? {}).filter(([, color]) => color));

// Settings saved before a field existed lack it, and older ones may name a font that is gone
const withDefaults = (a: Partial<Appearance>): Appearance => ({
	...defaultAppearance,
	...a,
	font: a.font?.kind === 'custom' ? a.font : defaultAppearance.font,
	surfaces: {
		light: { ...defaultAppearance.surfaces.light, ...setColors(a.surfaces?.light) },
		dark: { ...defaultAppearance.surfaces.dark, ...setColors(a.surfaces?.dark) }
	}
});

const appConfigService = new AppConfigService();
const config = fromStore(appConfigStore);

const objectUrls = new WeakMap<File, string>();
const objectUrl = (file: File) => {
	let url = objectUrls.get(file);
	if (!url) objectUrls.set(file, (url = URL.createObjectURL(file)));
	return url;
};

// The app shows the saved appearance, the draft only shows inside .appearance-draft previews until the unsaved-changes bar saves it
class AppearanceDraft {
	value = $state<Appearance>(withDefaults({}));
	// A file to upload, null to reset or remove the image, missing while unchanged
	images = $state<Partial<Record<BrandImageKey, File | null>>>({});
	// Bumped on discard so image fields that keep their own preview can remount
	imagesKey = $state(0);
	// The last font uploaded into the draft, which stays on offer while another font is picked
	fontFile = $state<File>();

	saved = $derived(withDefaults(config.current?.appearance ?? {}));

	changes = $derived(
		(Object.keys(defaultAppearance) as (keyof Appearance)[]).filter(
			(key) => !deepEqual(this.value[key], this.saved[key])
		).length +
			Object.keys(this.images).length +
			(this.value.font.kind === 'custom' && this.fontFile ? 1 : 0)
	);

	// The custom font to offer: the one uploaded into the draft, or else the saved one
	customFont = $derived.by(() => {
		if (this.fontFile) return { name: fontName(this.fontFile.name), url: objectUrl(this.fontFile) };
		if (this.saved.font.kind === 'custom')
			return { name: this.saved.font.name, url: cachedTitleFont.getUrl() };
		return null;
	});

	// The URL to show for an image: the pending file, the saved image, or null once it is removed
	imageUrl(key: BrandImageKey) {
		const file = this.images[key];
		if (file === null) return null;
		if (!file) return brandImages.find((image) => image.key === key)!.url();
		return objectUrl(file);
	}

	setImage(key: BrandImageKey, file: File | null | undefined) {
		if (file === undefined) delete this.images[key];
		else this.images[key] = file;
	}

	setFont(kind: Font['kind']) {
		this.value.font =
			kind === 'custom' && this.customFont
				? { kind: 'custom', name: this.customFont.name }
				: { kind: 'serif' };
	}

	uploadFont(file: File) {
		this.fontFile = file;
		this.setFont('custom');
	}

	// The files go first, so no one gets an appearance whose font or images are still missing
	async save(update: (config: Partial<AllAppConfig>) => Promise<void>) {
		const { logoLight, logoDark, background, emailLogo, favicon, defaultProfilePicture } =
			this.images;
		const font =
			this.value.font.kind === 'custom'
				? this.fontFile
				: this.saved.font.kind === 'custom'
					? null
					: undefined;
		await Promise.all([
			font === null ? appConfigService.deleteFont() : font && appConfigService.updateFont(font),
			favicon && appConfigService.updateFavicon(favicon),
			emailLogo && appConfigService.updateEmailLogo(emailLogo),
			logoLight === null
				? appConfigService.deleteLogo(true)
				: logoLight && appConfigService.updateLogo(logoLight, true),
			logoDark === null
				? appConfigService.deleteLogo(false)
				: logoDark && appConfigService.updateLogo(logoDark, false),
			defaultProfilePicture === null
				? appConfigService.deleteDefaultProfilePicture()
				: defaultProfilePicture &&
					appConfigService.updateDefaultProfilePicture(defaultProfilePicture),
			background === null
				? appConfigService.deleteBackgroundImage()
				: background && appConfigService.updateBackgroundImage(background)
		]);
		await update({ appearance: $state.snapshot(this.value) });
		this.images = {};
		this.fontFile = undefined;
	}

	discard() {
		this.value = structuredClone(this.saved);
		this.images = {};
		this.fontFile = undefined;
		this.imagesKey++;
	}
}

export const appearance = new AppearanceDraft();

// The previews switch between light and dark on their own, so the app keeps its mode while an admin compares both
class PreviewMode {
	#picked = $state<Mode | null>(null);

	// It follows the app's mode until a preview's switch picks one
	get current(): Mode {
		return this.#picked ?? (mode.current === 'dark' ? 'dark' : 'light');
	}

	set current(value: Mode) {
		this.#picked = value;
	}
}

export const previewMode = new PreviewMode();

// The classes of an element that shows the draft in the preview mode, every such element needs both
export const previewClass = () => `appearance-draft ${previewMode.current}`;

// Registers the draft with the unsaved-changes bar for as long as the calling component lives
// Leaving the form takes saving or discarding the draft, so the form starts it from the saved appearance, which only arrives with the app config
export function trackAppearance(update: (config: Partial<AllAppConfig>) => Promise<void>) {
	appearance.discard();
	trackUnsavedSection(
		() => appearance.changes,
		() => appearance.save(update),
		() => appearance.discard()
	);
}

// Font files are usually named Family-Style, like BricolageGrotesque-VariableFont_wght.ttf, so the name is the part before the first hyphen with its words split apart
const fontName = (fileName: string) =>
	fileName
		.replace(/\.[^.]+$/, '')
		.split('-')[0]
		.replace(/([a-z])([A-Z])/g, '$1 $2')
		.replaceAll('_', ' ')
		.trim() || fileName;

// The theme's body text is 15px, and text gets 1% more letter spacing than the theme
const THEME_TEXT_SIZE = 15;
const TRACKING = 0.01;

const textTokens = [
	'display-l',
	'display-m',
	'display-s',
	'title',
	'heading',
	'body-l',
	'body',
	'control',
	'caption',
	'micro',
	'mono',
	'mono-s',
	'code-cell',
	'xs',
	'sm',
	'xl'
];
const radiusTokens = [
	'radius',
	'radius-xs',
	'radius-sm',
	'radius-md',
	'radius-lg',
	'radius-xl',
	'radius-2xl',
	'radius-3xl',
	'radius-4xl'
];

type ThemeBase = ReturnType<typeof readThemeBase>;

// Reads the theme's own type, sizes and radii, so it must run while no appearance rule is applied
function readThemeBase() {
	const style = getComputedStyle(document.documentElement);
	const read = (name: string) => style.getPropertyValue(`--${name}`).trim();
	return {
		fontDisplay: read('font-display'),
		radii: radiusTokens.map((r) => ({ token: r, value: read(r) })).filter((r) => r.value),
		text: textTokens.map((t) => ({
			token: t,
			size: read(`text-${t}`),
			leading: read(`text-${t}--line-height`),
			tracking: read(`text-${t}--letter-spacing`)
		}))
	};
}

// Moves every channel of a neutral grey by the same amount
const shade = (hex: string, by: number) =>
	'#' +
	hex
		.slice(1)
		.match(/../g)!
		.map((c) =>
			Math.min(255, Math.max(0, parseInt(c, 16) + by))
				.toString(16)
				.padStart(2, '0')
		)
		.join('');

// Only colors that differ from the theme's own override it, so the default light mode keeps the theme exactly
function surfaceVars(mode: Mode, { background, sidebar, surface }: Surfaces) {
	const theme = monoTheme.tokens[mode];
	const newSidebar = sidebar !== theme.sidebar;
	const newSurface = surface !== theme.surface;
	const vars: Record<string, string> = { 'shell-bg': background };
	if (newSidebar) vars.sidebar = sidebar;

	// Fills and borders become translucent ink, so they read the same on every surface
	if (newSidebar || newSurface) {
		const ink = mode === 'light' ? '0 0 0' : '255 255 255';
		const alpha =
			mode === 'light'
				? { muted: 0.05, hover: 0.067, row: 0.02, selected: 0.08, border: 0.09 }
				: { muted: 0.09, hover: 0.1, row: 0.047, selected: 0.12, border: 0.12 };
		Object.assign(vars, {
			'fill-muted': `rgb(${ink} / ${alpha.muted})`,
			'fill-hover': `rgb(${ink} / ${alpha.hover})`,
			'fill-skeleton': `rgb(${ink} / ${alpha.hover})`,
			'fill-row-hover': `rgb(${ink} / ${alpha.row})`,
			'fill-selected': `rgb(${ink} / ${alpha.selected})`,
			border: `rgb(${ink} / ${alpha.border})`
		});
	}

	// Canvas follows the surface because form separators and calendars paint it to blend into the page
	if (newSurface) {
		Object.assign(vars, {
			surface,
			canvas: surface,
			'surface-raised': mode === 'light' ? '#ffffff' : shade(surface, 18),
			'surface-sunken': shade(surface, mode === 'light' ? -10 : 10)
		});
		if (mode === 'dark') {
			Object.assign(vars, {
				inverse: shade(surface, 31),
				'inverse-hover': shade(surface, 46),
				'inverse-border': shade(surface, 51)
			});
		}
	}
	return vars;
}

// The fit checks contrast against the surface, so it uses the appearance's surface where one is set
function accentFor(a: Appearance, mode: Mode) {
	const surface = a.surfaces[mode].surface;
	const rgb = surface ? parseColor(surface)?.rgb : undefined;
	const inputs = rgb
		? { ...monoTheme.accentInputs[mode], surface: rgb, canvas: rgb }
		: monoTheme.accentInputs[mode];
	return deriveAccent(a.primary ?? 'default', inputs);
}

// The primary as a mode shows it, and the status color it may be mistaken for
export function derivedPrimary(a: Appearance, mode: Mode) {
	const derived = accentFor(a, mode);
	const vars = primaryVars(derived);
	return { fill: vars.primary, fillFg: vars['primary-fg'], nearSemantic: derived.nearSemantic };
}

// A preview neither inherits the saved appearance nor the page's mode, so it sets the theme's whole mode and the draft's colors on top
function previewColorVars(mode: Mode, a: Appearance) {
	return {
		...monoTheme.tokens[mode],
		...primaryVars(accentFor(a, mode)),
		...surfaceVars(mode, a.surfaces[mode])
	};
}

function savedColorVars(mode: Mode, a: Appearance) {
	return {
		...surfaceVars(mode, a.surfaces[mode]),
		...(a.primary ? primaryVars(accentFor(a, mode)) : {})
	};
}

function typeVars(a: Appearance, base: ThemeBase, full: boolean) {
	const scale = a.textSize / THEME_TEXT_SIZE;
	const vars: Record<string, string> = {};
	for (const t of base.text) {
		if (t.size) vars[`text-${t.token}`] = `calc(${t.size} * ${scale})`;
		if (t.leading) vars[`text-${t.token}--line-height`] = `calc(${t.leading} * ${scale})`;
		if (t.tracking) vars[`text-${t.token}--letter-spacing`] = `calc(${t.tracking} + ${TRACKING}em)`;
	}
	if (full || a.radius !== 1)
		for (const { token, value } of base.radii) vars[token] = `calc(${value} * ${a.radius})`;
	if (full) vars['appearance-radius'] = String(a.radius);

	// A custom font replaces the theme's Young Serif, and a preview goes back to it when the draft has none
	if (a.font.kind === 'custom')
		vars['font-display'] = full ? fonts[1].family : "'Appearance Custom', 'Young Serif', serif";
	else if (full) vars['font-display'] = base.fontDisplay;
	return vars;
}

const block = (vars: Record<string, string>) =>
	Object.entries(vars)
		.map(([k, v]) => `--${k}: ${v};`)
		.join(' ');

const fontFace = (family: string, src: string) =>
	`@font-face { font-family: '${family}'; src: url("${src}"); font-display: swap; }`;

// The mode selectors outrank the :root and .dark rules of theme.css
function savedCss(a: Appearance, base: ThemeBase) {
	return [
		`html:root { ${block(typeVars(a, base, false))} }`,
		`html:root:not(.dark) { ${block(savedColorVars('light', a))} }`,
		`html.dark:root { ${block(savedColorVars('dark', a))} }`,
		a.font.kind === 'custom' ? fontFace('Appearance Custom', cachedTitleFont.getUrl()) : ''
	].join('\n');
}

function draftCss(a: Appearance, base: ThemeBase, fontUrl: string | undefined) {
	return [
		`.appearance-draft { ${block(typeVars(a, base, true))} }`,
		...(['light', 'dark'] as const).map(
			(mode) =>
				`.appearance-draft.${mode} { ${block(previewColorVars(mode, a))} color: var(--fg); color-scheme: ${mode}; }`
		),
		a.font.kind === 'custom' && fontUrl ? fontFace('Appearance Draft', fontUrl) : ''
	].join('\n');
}

// Writes the saved appearance for the app and the draft for the previews for as long as the calling component lives
export function applyAppearance() {
	// A hot reload mounts the layout again before the old one is gone, so its sheets go first to read the theme's own values
	const sheet = (id: string) => {
		document.getElementById(id)?.remove();
		const el = document.createElement('style');
		el.id = id;
		return el;
	};
	const live = sheet('appearance-saved');
	const draft = sheet('appearance-draft');
	const base = readThemeBase();
	document.head.append(live, draft);

	$effect(() => {
		live.textContent = savedCss(appearance.saved, base);
	});
	$effect(() => {
		draft.textContent = draftCss(appearance.value, base, appearance.customFont?.url);
	});
	$effect(() => () => {
		live.remove();
		draft.remove();
	});
}
