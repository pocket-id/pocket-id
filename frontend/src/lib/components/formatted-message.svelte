<!-- Component to display messages from Paraglide with support for links in the format <link href="url">text</link>. -->
<!-- This gets redundant in the future, because the library will support this natively. https://github.com/opral/inlang-sdk/issues/240 -->
<script lang="ts">
	let {
		m
	}: {
		m: string;
	} = $props();

	interface MessagePart {
		type: 'text' | 'link';
		content: string;
		href?: string;
	}

	function parseMessage(content: string): MessagePart[] | string {
		// Regex to match only <link href="url">text</link> format
		const linkRegex = /<link\s+href=(['"])(.*?)\1>(.*?)<\/link>/g;

		if (!linkRegex.test(content)) {
			return content;
		}

		// Reset regex lastIndex for reuse
		linkRegex.lastIndex = 0;

		const parts: MessagePart[] = [];
		let lastIndex = 0;
		let match;

		while ((match = linkRegex.exec(content)) !== null) {
			// Add text before the link
			if (match.index > lastIndex) {
				const textContent = content.slice(lastIndex, match.index);
				if (textContent) {
					parts.push({ type: 'text', content: textContent });
				}
			}

			const href = match[2];
			const linkText = match[3];

			parts.push({
				type: 'link',
				content: linkText,
				href: href
			});

			lastIndex = match.index + match[0].length;
		}

		// Add remaining text after the last link
		if (lastIndex < content.length) {
			const remainingText = content.slice(lastIndex);
			if (remainingText) {
				parts.push({ type: 'text', content: remainingText });
			}
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
