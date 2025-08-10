<!-- Component to display messages from Paraglide with support for links in the format <link href="url">text</link>. -->
<!-- This gets redundant in the future, because the library will support this natively. https://github.com/opral/inlang-sdk/issues/240 -->
<script lang="ts">
	let {
		m
	}: {
		m: string;
	} = $props();

	interface MessagePart {
		type: 'text' | 'link' | 'bold';
		content: string;
		href?: string;
	}

	// Extracts attribute value from a tag's attribute string
	function getAttr(attrs: string, name: string): string | undefined {
		const re = new RegExp(`\\b${name}\\s*=\\s*(["'])(.*?)\\1`, 'i');
		const m = re.exec(attrs ?? '');
		return m?.[2];
	}

	const handlers: Record<string, (attrs: string, inner: string) => MessagePart | null> = {
		link: (attrs, inner) => {
			const href = getAttr(attrs, 'href');
			if (!href) return { type: 'text', content: inner };
			return { type: 'link', content: inner, href };
		},
		b: (_attrs, inner) => ({ type: 'bold', content: inner })
	};

	function buildTokenRegex(): RegExp {
		const keys = Object.keys(handlers).join('|');
		// Matches: <tag attrs>inner</tag> for allowed tags only
		return new RegExp(`<(${keys})\\b([^>]*)>(.*?)<\\/\\1>`, 'g');
	}

	function parseMessage(content: string): MessagePart[] | string {
		const tokenRegex = buildTokenRegex();
		if (!tokenRegex.test(content)) return content;
		// Reset lastIndex for reuse
		tokenRegex.lastIndex = 0;

		const parts: MessagePart[] = [];
		let lastIndex = 0;
		let match: RegExpExecArray | null;

		while ((match = tokenRegex.exec(content)) !== null) {
			// Add text before the matched token
			if (match.index > lastIndex) {
				const textContent = content.slice(lastIndex, match.index);
				if (textContent) parts.push({ type: 'text', content: textContent });
			}

			const tag = match[1];
			const attrs = match[2] ?? '';
			const inner = match[3] ?? '';
			const handler = handlers[tag];
			const part: MessagePart | null = handler
				? handler(attrs, inner)
				: { type: 'text', content: inner };
			if (part) parts.push(part);

			lastIndex = match.index + match[0].length;
		}

		// Add remaining text after the last token
		if (lastIndex < content.length) {
			const remainingText = content.slice(lastIndex);
			if (remainingText) parts.push({ type: 'text', content: remainingText });
		}

		return parts;
	}

	const parsedContent = parseMessage(m);
</script>

{#if typeof parsedContent === 'string'}
	{parsedContent}
{:else}
	{#each parsedContent as part}
		{#if part.type === 'text'}
			{part.content}
		{:else if part.type === 'bold'}
			<b>
				{part.content}
			</b>
		{:else if part.type === 'link'}
			<a
				class="text-black underline dark:text-white"
				href={part.href}
				target="_blank"
				rel="noopener noreferrer"
			>
				{part.content}
			</a>
		{/if}
	{/each}
{/if}
