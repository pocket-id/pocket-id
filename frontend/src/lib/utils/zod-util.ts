import z from 'zod/v4';

export const optionalString = z
	.string()
	.transform((v) => (v === '' ? undefined : v))
	.optional();

export const optionalUrl = z
	.url()
	.optional()
	.or(z.literal('').transform(() => undefined));
