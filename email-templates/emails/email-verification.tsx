import { Link } from 'react-email';
import { BaseTemplate } from '../components/base-template';
import { Button } from '../components/button';
import CardHeader from '../components/card-header';
import { Muted, Paragraph } from '../components/text';
import { colors } from '../components/theme';
import type { Locale } from '../locales';
import { type SharedProps, sharedPreviewProps, sharedTemplateProps } from '../props';

interface EmailVerificationData {
	userFullName: string;
	verificationLink: string;
}

interface EmailVerificationProps extends SharedProps {
	data: EmailVerificationData;
	locale: Locale;
}

const strings: Record<
	Locale,
	{
		preview: (appName: string) => string;
		title: string;
		hello: (name: string) => string;
		body: (appName: string) => string;
		button: string;
		muted: string;
	}
> = {
	en: {
		preview: (appName) => `Confirm the email address for your ${appName} account`,
		title: 'Verify your email address',
		hello: (name) => `Hello ${name},`,
		body: (appName) =>
			`Click the button below to confirm the email address for your ${appName} account. This link expires in 24 hours.`,
		button: 'Verify email address',
		muted: "Or if you don't like clicking buttons, open this link:"
	},
	fr: {
		preview: (appName) => `Confirmez l'adresse e-mail de votre compte ${appName}`,
		title: 'Vérifiez votre adresse e-mail',
		hello: (name) => `Bonjour ${name},`,
		body: (appName) =>
			`Cliquez sur le bouton ci-dessous pour confirmer l'adresse e-mail de votre compte ${appName}. Ce lien expire dans 24 heures.`,
		button: "Vérifier l'adresse e-mail",
		muted: 'Si vous préférez ne pas cliquer sur un bouton, ouvrez ce lien :'
	}
};

export const EmailVerification = ({ data, locale, ...props }: EmailVerificationProps) => {
	const t = strings[locale];

	return (
		<BaseTemplate {...props} locale={locale} preview={t.preview(props.appName)}>
			<CardHeader title={t.title} />
			<Paragraph>{t.hello(data.userFullName)}</Paragraph>
			<Paragraph>{t.body(props.appName)}</Paragraph>

			<Button href={data.verificationLink}>{t.button}</Button>

			<Muted style={{ marginTop: '32px' }}>
				{t.muted}
				<br />
				<Link href={data.verificationLink} style={linkStyle}>
					{data.verificationLink}
				</Link>
			</Muted>
		</BaseTemplate>
	);
};

export default EmailVerification;

const linkStyle = {
	color: colors.mutedForeground,
	textDecoration: 'underline',
	wordBreak: 'break-all' as const
};

EmailVerification.TemplateProps = (_locale: Locale) => ({
	...sharedTemplateProps,
	data: {
		userFullName: '{{.Data.UserFullName}}',
		verificationLink: '{{.Data.VerificationLink}}'
	}
});

EmailVerification.PreviewProps = {
	...sharedPreviewProps,
	locale: 'en' as Locale,
	data: {
		userFullName: 'Tim Cook',
		verificationLink: 'https://id.example.com/user/verify-email?code=abcdefg12345'
	}
};
