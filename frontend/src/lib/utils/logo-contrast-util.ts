// The logo is judged on a small canvas, which still tells its shapes apart and keeps the scan cheap
const size = 48;
// A pixel with less contrast against black than this blends into a dark background
const minContrast = 2.5;
// The share of a logo that may blend into the background before it stops reading as that logo
const maxLostShare = 0.2;

// The linear value of an sRGB channel
function linear(channel: number) {
	const value = channel / 255;
	return value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4;
}

// Reports whether a logo stays recognizable on a dark background
// A logo that can't be loaded or read, for example because its host sends no CORS headers, counts as unrecognizable
export async function readsOnDarkBackground(url: string) {
	// Load the logo with CORS, because the pixels of an image loaded without it can't be read
	const image = new Image();
	image.crossOrigin = 'anonymous';
	image.src = url;
	try {
		await image.decode();
	} catch {
		return false;
	}

	// Scale the logo down and read its pixels
	const canvas = document.createElement('canvas');
	canvas.width = canvas.height = size;
	const context = canvas.getContext('2d')!;
	context.drawImage(image, 0, 0, size, size);
	const { data } = context.getImageData(0, 0, size, size);

	// Mark the pixels that blend into black, which includes the transparent ones, and count the pixels the logo is made of
	const faint = new Uint8Array(size * size);
	let logoPixels = 0;
	for (let i = 0; i < faint.length; i++) {
		const [r, g, b, a] = data.subarray(i * 4, i * 4 + 4);
		// Alpha scales the luminance linearly, because at this size it mostly says how much of a pixel an edge of the logo covers
		const luminance = (0.2126 * linear(r) + 0.7152 * linear(g) + 0.0722 * linear(b)) * (a / 255);
		faint[i] = Number((luminance + 0.05) / 0.05 < minContrast);
		if (a > 127) logoPixels++;
	}

	// Flood in from the edges through the faint pixels and count the logo pixels reached that way
	// A dark shape is only lost where nothing bright separates it from the background, so a dark glyph on a bright tile stays intact
	const stack: number[] = [];
	for (let i = 0; i < size; i++) {
		stack.push(i, size * (size - 1) + i, size * i, size * i + size - 1);
	}
	let lostPixels = 0;
	while (stack.length) {
		const i = stack.pop()!;
		if (!faint[i]) continue;
		// Clearing the mark keeps a pixel from being visited twice
		faint[i] = 0;
		if (data[i * 4 + 3] > 127) lostPixels++;

		const x = i % size;
		if (x > 0) stack.push(i - 1);
		if (x < size - 1) stack.push(i + 1);
		if (i >= size) stack.push(i - size);
		if (i < size * (size - 1)) stack.push(i + size);
	}

	// An image without any opaque pixels divides zero by zero, which fails the comparison like an unreadable logo
	return lostPixels / logoPixels <= maxLostShare;
}
