import { finalLogoPath, logoViewBox } from '#lib/components/logo.svelte';
import { cachedApplicationLogo } from '#lib/utils/cached-asset-util.ts';
import QRCode from 'qrcode';

// The logo's opacity at a fixed resolution, so modules can be cleared along its outline instead of in a box
export type LogoMask = { res: number; alpha: Uint8ClampedArray };

let logoMask: LogoMask | undefined;
let logoMaskLoading: Promise<LogoMask> | undefined;

export function cachedLogoMask() {
	return logoMask;
}

// Uses the same logo the code shows: the uploaded light logo, otherwise the Pocket ID mark
export function loadLogoMask() {
	const fallback = `data:image/svg+xml,${encodeURIComponent(
		`<svg xmlns="http://www.w3.org/2000/svg" width="713" height="713" viewBox="${logoViewBox}"><path d="${finalLogoPath}"/></svg>`
	)}`;
	logoMaskLoading ??= loadImage(cachedApplicationLogo.getUrl(true))
		.catch(() => loadImage(fallback))
		.then((image) => (logoMask = rasterize(image)));
	return logoMaskLoading;
}

function loadImage(src: string) {
	return new Promise<HTMLImageElement>((resolve, reject) => {
		const image = new Image();
		image.onload = () => resolve(image);
		image.onerror = reject;
		image.src = src;
	});
}

function rasterize(image: HTMLImageElement): LogoMask {
	const res = 96;
	const canvas = document.createElement('canvas');
	canvas.width = canvas.height = res;
	const ctx = canvas.getContext('2d', { willReadFrequently: true })!;
	// Matches the overlay's object-contain, so the mask lines up with the logo people see
	const scale = res / Math.max(image.naturalWidth, image.naturalHeight);
	const [w, h] = [image.naturalWidth * scale, image.naturalHeight * scale];
	ctx.drawImage(image, (res - w) / 2, (res - h) / 2, w, h);
	const pixels = ctx.getImageData(0, 0, res, res).data;
	const alpha = new Uint8ClampedArray(res * res);
	for (let i = 0; i < alpha.length; i++) alpha[i] = pixels[i * 4 + 3];
	return { res, alpha };
}

const f = (v: number) => +v.toFixed(3);

function circle(cx: number, cy: number, r: number) {
	return `M${f(cx - r)},${f(cy)}a${f(r)},${f(r)} 0 1 0 ${f(2 * r)},0a${f(r)},${f(r)} 0 1 0 ${f(-2 * r)},0Z`;
}

export function renderQr(value: string, mask?: LogoMask) {
	// The centred logo hides modules, so the code needs extra error correction to stay scannable
	const qr = QRCode.create(value, { errorCorrectionLevel: 'Q' });
	const n = qr.modules.size;
	const data = qr.modules.data;

	// With the logo's outline known, modules clear only where they would come close to it, so the dots wrap around the mark
	// Until the outline has loaded, a circle one module wider than the logo is cleared instead
	const logoSize = Math.round(n * (mask ? 0.26 : 0.2));
	const start = (n - logoSize) / 2;
	const margin = 0.45;
	const inLogo = (r: number, c: number) => {
		if (!mask) return Math.hypot(c + 0.5 - n / 2, r + 0.5 - n / 2) < logoSize / 2 + 1;
		const px = (v: number) => Math.round(((v - start) / logoSize) * mask.res);
		const [x0, x1] = [Math.max(0, px(c - margin)), Math.min(mask.res, px(c + 1 + margin))];
		const [y0, y1] = [Math.max(0, px(r - margin)), Math.min(mask.res, px(r + 1 + margin))];
		for (let y = y0; y < y1; y++) {
			for (let x = x0; x < x1; x++) {
				if (mask.alpha[y * mask.res + x] > 64) return true;
			}
		}
		return false;
	};
	const inEye = (r: number, c: number) =>
		(r < 7 && c < 7) || (r < 7 && c >= n - 7) || (r >= n - 7 && c < 7);

	let modules = '';
	for (let r = 0; r < n; r++) {
		for (let c = 0; c < n; c++) {
			if (data[r * n + c] && !inEye(r, c) && !inLogo(r, c)) {
				modules += circle(c + 0.5, r + 0.5, 0.36);
			}
		}
	}

	// Each eye is three concentric circles filled even-odd, so the middle one becomes the light ring
	let eyes = '';
	for (const [ox, oy] of [
		[0, 0],
		[n - 7, 0],
		[0, n - 7]
	]) {
		for (const radius of [3.5, 2.5, 1.5]) eyes += circle(ox + 3.5, oy + 3.5, radius);
	}

	return { n, modules, eyes, logo: { start, size: logoSize } };
}
