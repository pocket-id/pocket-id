<script lang="ts" generics="TMessage extends AnyMessage">
	import {
		ParaglideMessage,
		type MarkupRendererProps,
		type MessageLike,
		type MessageOptions
	} from '@inlang/paraglide-js-svelte';
	import type { Component, Snippet } from 'svelte';
	import type { AnyMessage, FormattedMessageProps } from './formatted-message';

	type BoldTag = {
		options: Record<string, never>;
		attributes: Record<string, never>;
		children: true;
	};

	type LinkTag = {
		options: { href: string };
		attributes: Record<string, never>;
		children: true;
	};

	type NativeMessageProps = {
		message: MessageLike<any, any, any>;
		inputs?: unknown;
		b: Snippet<[MarkupRendererProps<unknown, MessageOptions, BoldTag>]>;
		link: Snippet<[MarkupRendererProps<unknown, MessageOptions, LinkTag>]>;
	};

	const NativeParaglideMessage = ParaglideMessage as unknown as Component<NativeMessageProps>;

	let { message, inputs }: FormattedMessageProps<TMessage> = $props();
</script>

<NativeParaglideMessage {message} {inputs}>
	{#snippet b({ children })}
		<b>{@render children?.()}</b>
	{/snippet}
	{#snippet link({ children, options })}
		<a
			class="text-black underline dark:text-white"
			href={options.href}
			target="_blank"
			rel="noopener noreferrer"
		>
			{@render children?.()}
		</a>
	{/snippet}
</NativeParaglideMessage>
