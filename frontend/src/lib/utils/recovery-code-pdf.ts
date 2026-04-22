// Minimal hand-rolled PDF generator for the recovery-code emergency kit.
//
// We deliberately avoid pulling a large PDF dependency for what is, at the end
// of the day, a single printable page. The goal here is robustness and a clean
// minimalist layout, not fancy typography.
//
// Privacy: the generated document intentionally carries no identifying
// information about the instance or the account. A found or lost copy must not
// be traceable back to where it should be redeemed or for whom.

// --- Public API -----------------------------------------------------------

export type RecoveryKitPdfInput = {
	title: string;
	subtitle: string;
	introParagraphs: string[];
	codesHeading: string;
	codes: string[];
	warningTitle: string;
	warningParagraphs: string[];
};

export function buildRecoveryKitPdf(input: RecoveryKitPdfInput): Blob {
	const page = new PageBuilder();
	layoutKit(page, input);
	return page.toBlob();
}

export function downloadBlob(blob: Blob, filename: string): void {
	const url = URL.createObjectURL(blob);
	const link = document.createElement('a');
	link.href = url;
	link.download = filename;
	document.body.appendChild(link);
	link.click();
	document.body.removeChild(link);
	setTimeout(() => URL.revokeObjectURL(url), 1000);
}

// --- Layout ---------------------------------------------------------------

const PAGE_WIDTH = 612; // US Letter in points
const PAGE_HEIGHT = 792;
const MARGIN_X = 64;
const MARGIN_TOP = 72;
const CONTENT_WIDTH = PAGE_WIDTH - MARGIN_X * 2;

const BLACK: RGB = [0, 0, 0];
const MUTED: RGB = [0.38, 0.4, 0.45];
const RULE: RGB = [0.82, 0.83, 0.86];

function layoutKit(page: PageBuilder, input: RecoveryKitPdfInput): void {
	// Title
	page.text(input.title, { font: 'bold', size: 26, color: BLACK });
	page.advance(6);

	// Subtitle (small caps effect via UPPERCASE + letter-spacing via extra space)
	page.text(input.subtitle.toUpperCase(), {
		font: 'regular',
		size: 10,
		color: MUTED,
		tracking: 1.5
	});
	page.advance(18);

	page.rule(RULE);
	page.advance(18);

	// Intro paragraphs
	for (const p of input.introParagraphs) {
		page.paragraph(p, { font: 'regular', size: 11, color: BLACK, leading: 1.45 });
		page.advance(8);
	}

	page.advance(6);

	// Codes section heading
	page.text(input.codesHeading.toUpperCase(), {
		font: 'bold',
		size: 9,
		color: MUTED,
		tracking: 1.5
	});
	page.advance(14);

	// Codes in two columns, centred within the content area.
	page.codeColumns(input.codes, { size: 13, color: BLACK });

	page.advance(16);
	page.rule(RULE);
	page.advance(18);

	// Warning section
	page.text(input.warningTitle.toUpperCase(), {
		font: 'bold',
		size: 9,
		color: MUTED,
		tracking: 1.5
	});
	page.advance(10);

	for (const p of input.warningParagraphs) {
		page.paragraph(p, { font: 'regular', size: 10.5, color: MUTED, leading: 1.4 });
		page.advance(6);
	}
}

// --- Page builder ---------------------------------------------------------

type RGB = [number, number, number];

type TextOpts = {
	font: 'regular' | 'bold' | 'mono';
	size: number;
	color: RGB;
	tracking?: number; // extra points between glyphs (used for small caps look)
	leading?: number; // multiplier for line height in paragraphs
};

// Approximate character widths in points at size 1. These are conservative
// averages used only for soft-wrapping paragraphs — we do not need perfect
// metrics, only "does this fit within the content area".
const AVG_CHAR_WIDTH = { regular: 0.5, bold: 0.54, mono: 0.6 } as const;

const FONT_RES = { regular: 'F1', bold: 'F2', mono: 'F3' } as const;

class PageBuilder {
	private ops: string[] = [];
	private y = PAGE_HEIGHT - MARGIN_TOP;
	private currentFill: RGB = [-1, -1, -1];
	private currentStroke: RGB = [-1, -1, -1];

	// Move the cursor down by `pts` points.
	advance(pts: number): void {
		this.y -= pts;
	}

	// Draw a single line of text at the current cursor, left-aligned.
	text(value: string, opts: TextOpts & { x?: number }): void {
		const x = opts.x ?? MARGIN_X;
		// Baseline is the cursor. Move cursor down by full em height + 2 pts
		// descender so the next line does not overlap.
		this.setFill(opts.color);
		this.ops.push('BT');
		this.ops.push(`/${FONT_RES[opts.font]} ${opts.size} Tf`);
		if (opts.tracking) this.ops.push(`${opts.tracking} Tc`);
		this.ops.push(`1 0 0 1 ${fmt(x)} ${fmt(this.y)} Tm`);
		this.ops.push(`(${escapePdfText(value)}) Tj`);
		if (opts.tracking) this.ops.push(`0 Tc`);
		this.ops.push('ET');
		this.y -= opts.size; // advance past the glyph height
	}

	// Flow a paragraph as word-wrapped lines at `leading` multiple of size.
	paragraph(value: string, opts: TextOpts): void {
		const leading = opts.leading ?? 1.4;
		const avg = AVG_CHAR_WIDTH[opts.font];
		const maxChars = Math.max(20, Math.floor(CONTENT_WIDTH / (opts.size * avg)));
		const lines = wrap(value, maxChars);
		this.setFill(opts.color);
		for (let i = 0; i < lines.length; i++) {
			this.ops.push('BT');
			this.ops.push(`/${FONT_RES[opts.font]} ${opts.size} Tf`);
			this.ops.push(`1 0 0 1 ${fmt(MARGIN_X)} ${fmt(this.y)} Tm`);
			this.ops.push(`(${escapePdfText(lines[i])}) Tj`);
			this.ops.push('ET');
			this.y -= opts.size * leading;
		}
	}

	// Render a list of codes in two equal columns, centred across the page.
	codeColumns(codes: string[], opts: { size: number; color: RGB }): void {
		if (codes.length === 0) return;
		const halfway = Math.ceil(codes.length / 2);
		const colWidth = CONTENT_WIDTH / 2;
		// Estimate width of a code in monospace so we can centre each column.
		const codeWidth = (codes[0]?.length ?? 0) * opts.size * AVG_CHAR_WIDTH.mono;
		const col1X = MARGIN_X + (colWidth - codeWidth) / 2;
		const col2X = MARGIN_X + colWidth + (colWidth - codeWidth) / 2;
		const lineHeight = opts.size * 1.9;

		this.setFill(opts.color);
		for (let i = 0; i < halfway; i++) {
			const left = codes[i];
			const right = codes[i + halfway];
			// Left column
			this.ops.push('BT');
			this.ops.push(`/${FONT_RES.mono} ${opts.size} Tf`);
			this.ops.push(`1 0 0 1 ${fmt(col1X)} ${fmt(this.y)} Tm`);
			this.ops.push(`(${escapePdfText(left)}) Tj`);
			this.ops.push('ET');
			if (right !== undefined) {
				this.ops.push('BT');
				this.ops.push(`/${FONT_RES.mono} ${opts.size} Tf`);
				this.ops.push(`1 0 0 1 ${fmt(col2X)} ${fmt(this.y)} Tm`);
				this.ops.push(`(${escapePdfText(right)}) Tj`);
				this.ops.push('ET');
			}
			this.y -= lineHeight;
		}
	}

	// Horizontal divider at the current cursor.
	rule(color: RGB): void {
		this.setStroke(color);
		this.ops.push('0.5 w');
		this.ops.push(`${fmt(MARGIN_X)} ${fmt(this.y)} m`);
		this.ops.push(`${fmt(MARGIN_X + CONTENT_WIDTH)} ${fmt(this.y)} l`);
		this.ops.push('S');
	}

	private setFill(c: RGB): void {
		if (
			c[0] === this.currentFill[0] &&
			c[1] === this.currentFill[1] &&
			c[2] === this.currentFill[2]
		)
			return;
		this.currentFill = c;
		this.ops.push(`${fmt(c[0])} ${fmt(c[1])} ${fmt(c[2])} rg`);
	}

	private setStroke(c: RGB): void {
		if (
			c[0] === this.currentStroke[0] &&
			c[1] === this.currentStroke[1] &&
			c[2] === this.currentStroke[2]
		)
			return;
		this.currentStroke = c;
		this.ops.push(`${fmt(c[0])} ${fmt(c[1])} ${fmt(c[2])} RG`);
	}

	toBlob(): Blob {
		const content = this.ops.join('\n') + '\n';
		const pdf = encodePdf(content);
		return new Blob([pdf], { type: 'application/pdf' });
	}
}

// --- PDF wire format ------------------------------------------------------

function encodePdf(content: string): Uint8Array<ArrayBuffer> {
	const encoder = new TextEncoder();
	const contentBytes = encoder.encode(content);

	const objects: string[] = [
		'<< /Type /Catalog /Pages 2 0 R >>',
		'<< /Type /Pages /Kids [3 0 R] /Count 1 >>',
		`<< /Type /Page /Parent 2 0 R /MediaBox [0 0 ${PAGE_WIDTH} ${PAGE_HEIGHT}] ` +
			'/Contents 4 0 R ' +
			'/Resources << /Font << /F1 5 0 R /F2 6 0 R /F3 7 0 R >> >> >>',
		`<< /Length ${contentBytes.length} >>\nstream\n${content}endstream`,
		'<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>',
		'<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>',
		'<< /Type /Font /Subtype /Type1 /BaseFont /Courier-Bold /Encoding /WinAnsiEncoding >>'
	];

	let out = '%PDF-1.4\n%\xE2\xE3\xCF\xD3\n';
	const offsets: number[] = [];
	for (let i = 0; i < objects.length; i++) {
		offsets.push(out.length);
		out += `${i + 1} 0 obj\n${objects[i]}\nendobj\n`;
	}
	const xrefStart = out.length;
	out += `xref\n0 ${objects.length + 1}\n`;
	out += '0000000000 65535 f \n';
	for (const off of offsets) {
		out += off.toString().padStart(10, '0') + ' 00000 n \n';
	}
	// No /Info dictionary is emitted — this is deliberate. We do not want to
	// embed Author / Creator / Producer fields that could leak tooling or
	// identity information with the PDF.
	out += `trailer\n<< /Size ${objects.length + 1} /Root 1 0 R >>\nstartxref\n${xrefStart}\n%%EOF\n`;
	return encoder.encode(out);
}

// --- Helpers --------------------------------------------------------------

function escapePdfText(s: string): string {
	// Escape PDF string delimiters and backslashes. Also strip characters that
	// WinAnsiEncoding cannot represent so we never emit invalid bytes.
	return (
		s
			.replace(/\\/g, '\\\\')
			.replace(/\(/g, '\\(')
			.replace(/\)/g, '\\)')
			// Strip control characters except tab; keep printable + common latin-1.
			// eslint-disable-next-line no-control-regex
			.replace(/[\x00-\x08\x0B-\x1F\x7F]/g, '')
	);
}

function wrap(text: string, maxChars: number): string[] {
	// Respect explicit newlines the caller included.
	const paragraphs = text.split('\n');
	const out: string[] = [];
	for (const p of paragraphs) {
		if (p.length <= maxChars) {
			out.push(p);
			continue;
		}
		const words = p.split(/\s+/);
		let current = '';
		for (const word of words) {
			if (!current) {
				current = word;
				continue;
			}
			if (current.length + 1 + word.length > maxChars) {
				out.push(current);
				current = word;
			} else {
				current += ' ' + word;
			}
		}
		if (current) out.push(current);
	}
	return out;
}

function fmt(n: number): string {
	// Trim floats so the content stream stays compact and readable.
	return Number.isInteger(n) ? `${n}` : n.toFixed(3).replace(/\.?0+$/, '');
}
