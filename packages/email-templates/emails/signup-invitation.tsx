import { BaseTemplate } from '../components/base-template';
import { Button } from '../components/button';
import CardHeader from '../components/card-header';
import { Muted, MutedLink } from '../components/text';
import { type SharedProps, sharedPreviewProps, sharedTemplateProps } from '../props';

interface SignupInvitationData {
	signupLink: string;
	expirationString: string;
}

interface SignupInvitationProps extends SharedProps {
	data: SignupInvitationData;
}

export const SignupInvitation = ({ data, ...props }: SignupInvitationProps) => (
	<BaseTemplate {...props} preview={`You're invited to create an account on ${props.appName}`}>
		<CardHeader title="You're invited">
			Create your account on {props.appName} with the link below. It expires in{' '}
			{data.expirationString}.
		</CardHeader>

		<Button href={data.signupLink}>Create account</Button>

		<Muted>
			If the button doesn't work, open this link:
			<br />
			<MutedLink href={data.signupLink}>{data.signupLink}</MutedLink>
		</Muted>

		<Muted>If you weren't expecting this invitation, you can ignore this email.</Muted>
	</BaseTemplate>
);

export default SignupInvitation;

SignupInvitation.TemplateProps = {
	...sharedTemplateProps,
	data: {
		signupLink: '{{.Data.SignupLink}}',
		expirationString: '{{.Data.ExpirationString}}'
	}
};

SignupInvitation.PreviewProps = {
	...sharedPreviewProps,
	data: {
		signupLink: 'https://id.example.com/st/abcdefgh12345678',
		expirationString: '1 day'
	}
};
