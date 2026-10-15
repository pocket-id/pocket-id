// The languages supported by the instance-level "Email language" setting
// (Admin -> Application Configuration -> Email). Keep in sync with the
// EmailLanguageEnglish/EmailLanguageFrench constants in
// backend/internal/appconfig/model.go.
export type Locale = 'en' | 'fr';

export const locales: Locale[] = ['en', 'fr'];

export const defaultLocale: Locale = 'en';
