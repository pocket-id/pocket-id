import type { MessageLike } from '@inlang/paraglide-js-svelte';

export type AnyMessage = MessageLike<any, any, any>;

type MessageInputs<TMessage extends AnyMessage> = Parameters<TMessage>[0];

type MessageInputsProp<TInputs> = undefined extends TInputs
	? { inputs?: Exclude<NoInfer<TInputs>, undefined> }
	: { inputs: NoInfer<TInputs> };

export type FormattedMessageProps<TMessage extends AnyMessage> = {
	message: TMessage;
} & MessageInputsProp<MessageInputs<TMessage>>;

export type AnyFormattedMessage = {
	message: AnyMessage;
	inputs?: unknown;
};
