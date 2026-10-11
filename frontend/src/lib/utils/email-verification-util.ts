import type { AppConfig } from '#lib/types/application-configuration.type.ts';
import type { User } from '#lib/types/user.type.ts';

// Reports whether the user must verify their email before they can use their account
// Mirrors the backend rule: only regular users are restricted, and only while verification is required
export function isEmailVerificationPending(
	user: User | null | undefined,
	appConfig: AppConfig | null | undefined
) {
	return (
		!!user && appConfig?.emailVerification === 'required' && !user.isAdmin && !user.emailVerified
	);
}
