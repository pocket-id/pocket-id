<script lang="ts" generics="TMessage extends AnyMessage">
	import type { AnyMessage, FormattedMessageProps } from './formatted-message';

	let { message, inputs, links }: FormattedMessageProps<TMessage> = $props();

	// Private-use characters that never appear in a message
	const MARK_START = String.fromCharCode(0xe000);
	const MARK_END = String.fromCharCode(0xe001);
	const MARKER = new RegExp(`${MARK_START}(\\d+)${MARK_END}`, 'g');

	// Messages mark bold text with <b>…</b> and a link with a tag named after a key of links, e.g. <license>CC BY 4.0</license>
	const segments = $derived.by(() => {
		// String inputs are swapped for private-use markers while formatting, so a value that contains a tag is never read as markup
		const values: string[] = [];
		const masked = Object.fromEntries(
			Object.entries(inputs ?? {}).map(([key, value]) => {
				if (typeof value !== 'string') return [key, value];
				values.push(value);
				return [key, `${MARK_START}${values.length - 1}${MARK_END}`];
			})
		);
		const unmask = (text: string) => text.replace(MARKER, (_, index) => values[Number(index)]);

		// Split the formatted text into plain runs and tagged runs
		const text = message(masked);
		const result: { tag?: string; text: string }[] = [];
		let last = 0;
		for (const match of text.matchAll(/<([a-z]+)>([\s\S]*?)<\/\1>/g)) {
			if (match.index > last) result.push({ text: unmask(text.slice(last, match.index)) });
			result.push({ tag: match[1], text: unmask(match[2]) });
			last = match.index + match[0].length;
		}
		if (last < text.length) result.push({ text: unmask(text.slice(last)) });
		return result;
	});
</script>

{#each segments as segment, index (index)}
	{#if segment.tag === 'b'}
		<b>{segment.text}</b>
	{:else if segment.tag && links?.[segment.tag]}
		<a class="inline-link" href={links[segment.tag]} target="_blank" rel="noopener noreferrer">
			{segment.text}
		</a>
	{:else}
		{segment.text}
	{/if}
{/each}
