import { Link } from 'react-email';
import { BaseTemplate } from '../components/base-template';
import { Button } from '../components/button';
import CardHeader from '../components/card-header';
import { CodeBox } from '../components/code-box';
import { Muted, Paragraph } from '../components/text';
import { colors } from '../components/theme';
import type { Locale } from '../locales';
import { type SharedProps, sharedPreviewProps, sharedTemplateProps } from '../props';

interface OneTimeAccessData {
	code: string;
	loginLink: string;
	buttonCodeLink: string;
	expirationString: string;
}

interface OneTimeAccessEmailProps extends SharedProps {
	data: OneTimeAccessData;
	locale: Locale;
}

const strings: Record<
	Locale,
	{
		preview: (appName: string, code: string) => string;
		title: string;
		body: (appName: string, expirationString: string) => string;
		button: string;
		mutedPrefix: string;
		mutedSuffix: string;
	}
> = {
	en: {
		preview: (appName, code) => `Your ${appName} login code is ${code}`,
		title: 'Your login code',
		body: (appName, expirationString) =>
			`Use the code below to sign in to ${appName}. It expires in ${expirationString}.`,
		button: 'Sign in',
		mutedPrefix: 'Or open',
		mutedSuffix: 'and enter the code manually.'
	},
	fr: {
		preview: (appName, code) => `Votre code de connexion ${appName} est ${code}`,
		title: 'Votre code de connexion',
		body: (appName, expirationString) =>
			`Utilisez le code ci-dessous pour vous connecter à ${appName}. Il expire dans ${expirationString}.`,
		button: 'Se connecter',
		mutedPrefix: 'Ou ouvrez',
		mutedSuffix: 'et saisissez le code manuellement.'
	}
};

export const OneTimeAccessEmail = ({ data, locale, ...props }: OneTimeAccessEmailProps) => {
	const t = strings[locale];

	return (
		<BaseTemplate {...props} locale={locale} preview={t.preview(props.appName, data.code)}>
			<CardHeader title={t.title} />
			<Paragraph>{t.body(props.appName, data.expirationString)}</Paragraph>

			<CodeBox code={data.code} />

			<Button href={data.buttonCodeLink}>{t.button}</Button>

			<Muted style={{ marginTop: '32px' }}>
				{t.mutedPrefix}{' '}
				<Link href={data.loginLink} style={linkStyle}>
					{data.loginLink}
				</Link>{' '}
				{t.mutedSuffix}
			</Muted>
		</BaseTemplate>
	);
};

export default OneTimeAccessEmail;

const linkStyle = {
	color: colors.mutedForeground,
	textDecoration: 'underline',
	wordBreak: 'break-all' as const
};

OneTimeAccessEmail.TemplateProps = (_locale: Locale) => ({
	...sharedTemplateProps,
	data: {
		code: '{{.Data.Code}}',
		loginLink: '{{.Data.LoginLink}}',
		buttonCodeLink: '{{.Data.LoginLinkWithCode}}',
		expirationString: '{{.Data.ExpirationString}}'
	}
});

OneTimeAccessEmail.PreviewProps = {
	...sharedPreviewProps,
	locale: 'en' as Locale,
	data: {
		code: '123456',
		loginLink: 'https://id.example.com/lc',
		buttonCodeLink: 'https://id.example.com/lc/123456',
		expirationString: '15 minutes'
	}
};
