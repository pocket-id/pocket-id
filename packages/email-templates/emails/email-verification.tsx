import { BaseTemplate } from '../components/base-template';
import { Button } from '../components/button';
import CardHeader from '../components/card-header';
import { Muted, MutedLink } from '../components/text';
import { type SharedProps, sharedPreviewProps, sharedTemplateProps } from '../props';

interface EmailVerificationData {
	userFullName: string;
	verificationLink: string;
}

interface EmailVerificationProps extends SharedProps {
	data: EmailVerificationData;
}

export const EmailVerification = ({ data, ...props }: EmailVerificationProps) => (
	<BaseTemplate
		{...props}
		preview={props.t('email_verification_preview', { appName: props.appName })}
	>
		<CardHeader title={props.t('email_verification_title')}>
			{props.t('email_verification_description', {
				name: data.userFullName,
				appName: props.appName
			})}
		</CardHeader>

		<Button href={data.verificationLink}>{props.t('email_verification_button')}</Button>

		<Muted>
			{props.t('link_fallback')}
			<br />
			<MutedLink href={data.verificationLink}>{data.verificationLink}</MutedLink>
		</Muted>
	</BaseTemplate>
);

export default EmailVerification;

EmailVerification.TemplateProps = {
	...sharedTemplateProps,
	data: {
		userFullName: '{{.Data.UserFullName}}',
		verificationLink: '{{.Data.VerificationLink}}'
	}
};

EmailVerification.PreviewProps = {
	...sharedPreviewProps,
	data: {
		userFullName: 'Tim Cook',
		verificationLink: 'https://id.example.com/user/verify-email?code=abcdefg12345'
	}
};
