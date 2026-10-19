import { m } from '#lib/paraglide/messages.js';
import type { z } from 'zod/v4';

const isEmpty = (input: unknown) => input === undefined || input === null || input === '';

// Plain-language messages for the checks forms use; anything not covered falls back to Zod's own locale
export const validationErrorMap: z.core.$ZodErrorMap = (issue) => {
	switch (issue.code) {
		case 'invalid_type':
			if (isEmpty(issue.input)) return m.validation_required();
			if (issue.expected === 'number' || issue.expected === 'int') return m.validation_number();
			return undefined;
		case 'too_small': {
			const count = Number(issue.minimum);
			if (issue.origin === 'string') {
				return count <= 1 ? m.validation_required() : m.validation_min_length({ count });
			}
			if (issue.origin === 'number' || issue.origin === 'int') {
				return m.validation_min_number({ minimum: count });
			}
			if (issue.origin === 'array' || issue.origin === 'set')
				return m.validation_min_items({ count });
			return undefined;
		}
		case 'too_big': {
			const count = Number(issue.maximum);
			if (issue.origin === 'string') return m.validation_max_length({ count });
			if (issue.origin === 'number' || issue.origin === 'int') {
				return m.validation_max_number({ maximum: count });
			}
			if (issue.origin === 'array' || issue.origin === 'set')
				return m.validation_max_items({ count });
			return undefined;
		}
		case 'invalid_format':
			if (isEmpty(issue.input)) return m.validation_required();
			if (issue.format === 'email') return m.validation_email();
			if (issue.format === 'url') return m.validation_url();
			return undefined;
		default:
			return undefined;
	}
};
