import { BaseTemplate } from '../components/base-template';
import CardHeader from '../components/card-header';
import { Paragraph } from '../components/text';
import type { Locale } from '../locales';
import { type SharedProps, sharedPreviewProps, sharedTemplateProps } from '../props';

interface TestEmailProps extends SharedProps {
	locale: Locale;
}

const strings: Record<Locale, { preview: string; title: string; body: string }> = {
	en: {
		preview: 'Your email setup is working correctly',
		title: 'Test email',
		body: 'Your email setup is working correctly!'
	},
	fr: {
		preview: 'La configuration de votre messagerie fonctionne correctement',
		title: 'E-mail de test',
		body: 'La configuration de votre messagerie fonctionne correctement !'
	}
};

export const TestEmail = ({ locale, ...props }: TestEmailProps) => {
	const t = strings[locale];

	return (
		<BaseTemplate {...props} locale={locale} preview={t.preview}>
			<CardHeader title={t.title} />
			<Paragraph>{t.body}</Paragraph>
		</BaseTemplate>
	);
};

export default TestEmail;

TestEmail.TemplateProps = (_locale: Locale) => ({
	...sharedTemplateProps
});

TestEmail.PreviewProps = {
	...sharedPreviewProps,
	locale: 'en' as Locale
};
