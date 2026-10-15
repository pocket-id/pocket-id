import { BaseTemplate } from '../components/base-template';
import CardHeader from '../components/card-header';
import { Muted, Paragraph } from '../components/text';
import type { Locale } from '../locales';
import { type SharedProps, sharedPreviewProps, sharedTemplateProps } from '../props';

interface ApiKeyExpiringData {
	name: string;
	apiKeyName: string;
	expiresAt: string;
}

interface ApiKeyExpiringEmailProps extends SharedProps {
	data: ApiKeyExpiringData;
	locale: Locale;
}

const strings: Record<
	Locale,
	{
		preview: (apiKeyName: string, expiresAt: string) => string;
		title: string;
		hello: (name: string) => string;
		body: (apiKeyName: string, expiresAt: string) => [string, string, string];
		muted: (appName: string) => string;
	}
> = {
	en: {
		preview: (apiKeyName, expiresAt) => `Your API key ${apiKeyName} expires on ${expiresAt}`,
		title: 'API key expiring soon',
		hello: (name) => `Hello ${name},`,
		body: (apiKeyName, expiresAt) => [
			'Your API key ',
			apiKeyName,
			` will expire on ${expiresAt}. Anything that uses this key will stop working once it expires.`
		],
		muted: (appName) =>
			`To keep access, create a new API key in your ${appName} account settings before then.`
	},
	fr: {
		preview: (apiKeyName, expiresAt) => `Votre clé API ${apiKeyName} expire le ${expiresAt}`,
		title: 'Clé API bientôt expirée',
		hello: (name) => `Bonjour ${name},`,
		body: (apiKeyName, expiresAt) => [
			'Votre clé API ',
			apiKeyName,
			` expirera le ${expiresAt}. Tout ce qui utilise cette clé cessera de fonctionner une fois qu'elle aura expiré.`
		],
		muted: (appName) =>
			`Pour conserver l'accès, créez une nouvelle clé API dans les paramètres de votre compte ${appName} avant cette date.`
	}
};

export const ApiKeyExpiringEmail = ({ data, locale, ...props }: ApiKeyExpiringEmailProps) => {
	const t = strings[locale];
	const [bodyPrefix, apiKeyName, bodySuffix] = t.body(data.apiKeyName, data.expiresAt);

	return (
		<BaseTemplate {...props} locale={locale} preview={t.preview(data.apiKeyName, data.expiresAt)}>
			<CardHeader title={t.title} />
			<Paragraph>{t.hello(data.name)}</Paragraph>
			<Paragraph>
				{bodyPrefix}
				<strong>{apiKeyName}</strong>
				{bodySuffix}
			</Paragraph>

			<Muted>{t.muted(props.appName)}</Muted>
		</BaseTemplate>
	);
};

export default ApiKeyExpiringEmail;

ApiKeyExpiringEmail.TemplateProps = (_locale: Locale) => ({
	...sharedTemplateProps,
	data: {
		name: '{{.Data.Name}}',
		apiKeyName: '{{.Data.ApiKeyName}}',
		expiresAt: '{{.Data.ExpiresAt.Format "2006-01-02 15:04:05 MST"}}'
	}
});

ApiKeyExpiringEmail.PreviewProps = {
	...sharedPreviewProps,
	locale: 'en' as Locale,
	data: {
		name: 'Elias',
		apiKeyName: 'CI deploy key',
		expiresAt: '2026-01-30 12:00:00 UTC'
	}
};
