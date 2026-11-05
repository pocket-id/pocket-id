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
	<BaseTemplate {...props} preview={`Confirm the email address for your ${props.appName} account`}>
		<CardHeader title="Verify your email address">
			Hello {data.userFullName}, confirm that this is the email address for your {props.appName}{' '}
			account. The link expires in 24 hours.
		</CardHeader>

		<Button href={data.verificationLink}>Verify email address</Button>

		<Muted>
			If the button doesn't work, open this link:
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
