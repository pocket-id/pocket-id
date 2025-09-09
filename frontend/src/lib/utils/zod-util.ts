import { m } from '$lib/paraglide/messages';
import z from 'zod/v4';

export const emptyToUndefined = <T>(validation: z.ZodType<T>) =>
	z.preprocess((v) => (v === '' ? undefined : v), validation);

export const optionalUrl = z
	.url()
	.optional()
	.or(z.literal('').transform(() => undefined));

export const callbackUrlSchema = z
	.string()
	.nonempty()
	.refine(
		(val) => {
			try {
				new URL(val.replace(/\*/g, 'x'));
				return true;
			} catch {
				return false;
			}
		},
		{
			message: m.invalid_redirect_url()
		}
	);
