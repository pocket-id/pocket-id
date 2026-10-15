import { BaseTemplate } from '../components/base-template';
import CardHeader from '../components/card-header';
import { DetailsList } from '../components/details-list';
import { Muted, Paragraph } from '../components/text';
import type { Locale } from '../locales';
import { type SharedProps, sharedPreviewProps, sharedTemplateProps } from '../props';

interface SignInData {
	location: string;
	ipAddress: string;
	device: string;
	method: string;
	dateTime: string;
}

interface NewSignInEmailProps extends SharedProps {
	data: SignInData;
	locale: Locale;
}

const strings: Record<
	Locale,
	{
		preview: (appName: string) => string;
		title: string;
		body: (appName: string) => string;
		location: string;
		ipAddress: string;
		device: string;
		method: string;
		time: string;
		unknownLocation: string;
		muted: (appName: string) => string;
	}
> = {
	en: {
		preview: (appName) => `A new sign-in to your ${appName} account was detected`,
		title: 'New sign-in detected',
		body: (appName) =>
			`Your ${appName} account was recently accessed from a new IP address or browser. If this was you, no further action is needed.`,
		location: 'Approximate location',
		ipAddress: 'IP address',
		device: 'Device',
		method: 'Sign-in method',
		time: 'Time',
		unknownLocation: 'Unknown',
		muted: (appName) =>
			`If you don't recognize this activity, review the passkeys in your ${appName} account settings and remove any you don't recognize.`
	},
	fr: {
		preview: (appName) => `Une nouvelle connexion à votre compte ${appName} a été détectée`,
		title: 'Nouvelle connexion détectée',
		body: (appName) =>
			`Votre compte ${appName} a été accédé récemment depuis une nouvelle adresse IP ou un nouveau navigateur. Si c'est bien vous, aucune action n'est nécessaire.`,
		location: 'Localisation approximative',
		ipAddress: 'Adresse IP',
		device: 'Appareil',
		method: 'Méthode de connexion',
		time: 'Date et heure',
		unknownLocation: 'Inconnue',
		muted: (appName) =>
			`Si vous ne reconnaissez pas cette activité, vérifiez les clés d'accès de votre compte ${appName} et supprimez celles que vous ne reconnaissez pas.`
	}
};

export const NewSignInEmail = ({ data, locale, ...props }: NewSignInEmailProps) => {
	const t = strings[locale];

	return (
		<BaseTemplate {...props} locale={locale} preview={t.preview(props.appName)}>
			<CardHeader title={t.title} />
			<Paragraph>{t.body(props.appName)}</Paragraph>

			<DetailsList
				items={[
					{ label: t.location, value: data.location },
					{ label: t.ipAddress, value: data.ipAddress },
					{ label: t.device, value: data.device },
					{ label: t.method, value: data.method },
					{ label: t.time, value: data.dateTime }
				]}
			/>

			<Muted>{t.muted(props.appName)}</Muted>
		</BaseTemplate>
	);
};

export default NewSignInEmail;

NewSignInEmail.TemplateProps = (locale: Locale) => ({
	...sharedTemplateProps,
	data: {
		location: `{{if and .Data.City .Data.Country}}{{.Data.City}}, {{.Data.Country}}{{else if .Data.Country}}{{.Data.Country}}{{else}}${strings[locale].unknownLocation}{{end}}`,
		ipAddress: '{{.Data.IPAddress}}',
		device: '{{.Data.Device}}',
		method: '{{.Data.Method}}',
		dateTime: '{{.Data.DateTime.Format "January 2, 2006 at 3:04 PM MST"}}'
	}
});

NewSignInEmail.PreviewProps = {
	...sharedPreviewProps,
	locale: 'en' as Locale,
	data: {
		location: 'San Francisco, USA',
		ipAddress: '203.0.113.42',
		device: 'Chrome on macOS',
		method: 'Passkey',
		dateTime: 'January 2, 2026 at 3:04 PM UTC'
	}
};
