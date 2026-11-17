import { BaseTemplate } from '../components/base-template';
import { Button } from '../components/button';
import CardHeader from '../components/card-header';
import { CodeBox } from '../components/code-box';
import { Muted, MutedLink } from '../components/text';
import { type SharedProps, sharedPreviewProps, sharedTemplateProps } from '../props';

interface OneTimeAccessData {
	code: string;
	loginLink: string;
	buttonCodeLink: string;
	expiration: string;
}

interface OneTimeAccessEmailProps extends SharedProps {
	data: OneTimeAccessData;
}

export const OneTimeAccessEmail = ({ data, ...props }: OneTimeAccessEmailProps) => (
	<BaseTemplate
		{...props}
		preview={props.t('one_time_access_preview', { appName: props.appName, code: data.code })}
	>
		<CardHeader title={props.t('one_time_access_title')}>
			{props.t('one_time_access_description', {
				appName: props.appName,
				expiration: data.expiration
			})}
		</CardHeader>

		<CodeBox code={data.code} />

		<Button href={data.buttonCodeLink}>{props.t('one_time_access_button')}</Button>

		{/* The link shows the URL itself, so the plain-text version prints it only once */}
		<Muted>
			{props.t('one_time_access_manual', {}, 0)}
			<MutedLink href={data.loginLink}>{data.loginLink}</MutedLink>
			{props.t('one_time_access_manual', {}, 2)}
		</Muted>
	</BaseTemplate>
);

export default OneTimeAccessEmail;

OneTimeAccessEmail.TemplateProps = {
	...sharedTemplateProps,
	data: {
		code: '{{.Data.Code}}',
		loginLink: '{{.Data.LoginLink}}',
		buttonCodeLink: '{{.Data.LoginLinkWithCode}}',
		expiration: '{{.Duration .Data.Expiration}}'
	}
};

OneTimeAccessEmail.PreviewProps = {
	...sharedPreviewProps,
	data: {
		code: '123456',
		loginLink: 'https://id.example.com/lc',
		buttonCodeLink: 'https://id.example.com/lc/123456',
		expiration: '15 minutes'
	}
};
