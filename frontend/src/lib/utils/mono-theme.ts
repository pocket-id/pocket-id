// The neutrals of Pocket ID's one theme, which theme.css also declares, and what its primary is fitted against at runtime
import type { Theme } from './color-theme-util.ts';

export const monoTheme: Theme = {
	tokens: {
		light: {
			sidebar: '#fafafa',
			canvas: '#ffffff',
			'surface-sunken': '#f5f5f5',
			'fill-muted': '#f2f2f2',
			'fill-hover': '#eeeeee',
			'fill-row-hover': '#fafafa',
			'fill-selected': '#ebebeb',
			border: '#e8e8e8',
			'border-strong': '#8c8c8c',
			fg: '#0a0a0a',
			'fg-muted': '#5e5e5e',
			'fg-subtle': '#a3a3a3',
			inverse: '#111111',
			'inverse-fg': '#fafafa',
			'inverse-muted': '#b5b5b5',
			'inverse-hover': '#262626',
			surface: '#ffffff',
			'surface-raised': '#ffffff',
			'fill-skeleton': '#eeeeee',
			'border-control': '#8c8c8c',
			'info-solid': '#5e5e5e',
			'info-fg': '#0a0a0a',
			'info-soft': '#f5f5f5',
			'info-border': '#e8e8e8',
			'well-light': '#f5f5f5',
			'well-dark': '#000000',
			'well-light-fg': '#5e5e5e',
			'well-dark-fg': '#a1a1a1'
		},
		dark: {
			sidebar: '#0a0a0a',
			canvas: '#000000',
			'surface-sunken': '#0e0e0e',
			'fill-muted': '#171717',
			'fill-hover': '#1a1a1a',
			'fill-row-hover': '#0c0c0c',
			'fill-selected': '#1f1f1f',
			border: '#1f1f1f',
			'border-strong': '#6b6b6b',
			fg: '#ededed',
			'fg-muted': '#a1a1a1',
			'fg-subtle': '#5c5c5c',
			inverse: '#1f1f1f',
			'inverse-fg': '#ededed',
			'inverse-muted': '#b5b5b5',
			'inverse-hover': '#2e2e2e',
			'inverse-border': '#333333',
			surface: '#000000',
			'surface-raised': '#121212',
			'fill-skeleton': '#1a1a1a',
			'border-control': '#6b6b6b',
			'info-solid': '#a1a1a1',
			'info-fg': '#ededed',
			'info-soft': '#0e0e0e',
			'info-border': '#1f1f1f',
			'well-light': '#f5f5f5',
			'well-dark': '#000000',
			'well-light-fg': '#5e5e5e',
			'well-dark-fg': '#a1a1a1'
		}
	},
	accentInputs: {
		light: {
			name: 'light',
			surface: [255, 255, 255],
			canvas: [255, 255, 255],
			fg: [10, 10, 10],
			inkFill: [10, 10, 10],
			inkFillFg: [255, 255, 255],
			inkHover: [38, 38, 38],
			borderControl: [140, 140, 140],
			controlBgs: [
				[255, 255, 255],
				[255, 255, 255],
				[255, 255, 255],
				[245, 245, 245],
				[242, 242, 242],
				[238, 238, 238],
				[250, 250, 250],
				[235, 235, 235]
			],
			semanticHues: {
				danger: 25,
				warning: 65,
				success: 150
			}
		},
		dark: {
			name: 'dark',
			surface: [0, 0, 0],
			canvas: [0, 0, 0],
			fg: [237, 237, 237],
			inkFill: [237, 237, 237],
			inkFillFg: [0, 0, 0],
			inkHover: [214, 214, 214],
			borderControl: [107, 107, 107],
			controlBgs: [
				[0, 0, 0],
				[0, 0, 0],
				[18, 18, 18],
				[14, 14, 14],
				[23, 23, 23],
				[26, 26, 26],
				[12, 12, 12],
				[31, 31, 31]
			],
			semanticHues: {
				danger: 25,
				warning: 75,
				success: 150
			}
		}
	}
};
