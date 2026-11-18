export type AnyMessage = (inputs?: any) => string;

type MessageInputs<TMessage extends AnyMessage> = Parameters<TMessage>[0];

type MessageInputsProp<TInputs> = undefined extends TInputs
	? { inputs?: Exclude<NoInfer<TInputs>, undefined> }
	: { inputs: NoInfer<TInputs> };

export type FormattedMessageProps<TMessage extends AnyMessage> = {
	message: TMessage;
	// URLs of the link tags the message uses, by tag name
	links?: Record<string, string>;
} & MessageInputsProp<MessageInputs<TMessage>>;

export type AnyFormattedMessage = {
	message: AnyMessage;
	inputs?: unknown;
	links?: Record<string, string>;
};
