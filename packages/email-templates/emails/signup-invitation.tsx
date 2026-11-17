import { BaseTemplate } from '../components/base-template';
import { Button } from '../components/button';
import CardHeader from '../components/card-header';
import { Muted, MutedLink } from '../components/text';
import { type SharedProps, sharedPreviewProps, sharedTemplateProps } from '../props';

interface SignupInvitationData {
	signupLink: string;
	expiration: string;
}

interface SignupInvitationProps extends SharedProps {
	data: SignupInvitationData;
}

export const SignupInvitation = ({ data, ...props }: SignupInvitationProps) => (
	<BaseTemplate
		{...props}
		preview={props.t('signup_invitation_preview', { appName: props.appName })}
	>
		<CardHeader title={props.t('signup_invitation_title')}>
			{props.t('signup_invitation_description', {
				appName: props.appName,
				expiration: data.expiration
			})}
		</CardHeader>

		<Button href={data.signupLink}>{props.t('signup_invitation_button')}</Button>

		<Muted>
			{props.t('link_fallback')}
			<br />
			<MutedLink href={data.signupLink}>{data.signupLink}</MutedLink>
		</Muted>

		<Muted>{props.t('signup_invitation_ignore')}</Muted>
	</BaseTemplate>
);

export default SignupInvitation;

SignupInvitation.TemplateProps = {
	...sharedTemplateProps,
	data: {
		signupLink: '{{.Data.SignupLink}}',
		expiration: '{{.Duration .Data.Expiration}}'
	}
};

SignupInvitation.PreviewProps = {
	...sharedPreviewProps,
	data: {
		signupLink: 'https://id.example.com/st/abcdefgh12345678',
		expiration: '1 day'
	}
};
