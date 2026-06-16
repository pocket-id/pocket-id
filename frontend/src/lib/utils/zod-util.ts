import { m } from '$lib/paraglide/messages';
import { z } from 'zod/v4';

export const emptyToUndefined = <T>(validation: z.ZodType<T>) =>
	z.preprocess((v) => (v === '' ? undefined : v), validation.optional());

export const optionalUrl = z
	.url()
	.optional()
	.or(z.literal('').transform(() => undefined));


export const usernameSchema = z
	.string()
	.min(1)
	.max(30)
	.regex(/^[a-zA-Z0-9]/, m.username_must_start_with())
	.regex(/[a-zA-Z0-9]$/, m.username_must_end_with())
	.regex(/^[a-zA-Z0-9_.@-]+$/, m.username_can_only_contain());
