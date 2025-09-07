import z from 'zod/v4';

export const emptyToUndefined = <T>(validation: z.ZodType<T>) =>
	z.preprocess((v) => (v === '' ? undefined : v), validation.optional());

export const optionalUrl = z
	.url()
	.optional()
	.or(z.literal('').transform(() => undefined));
