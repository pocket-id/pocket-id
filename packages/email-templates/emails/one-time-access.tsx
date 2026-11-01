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
	expirationString: string;
}

interface OneTimeAccessEmailProps extends SharedProps {
	data: OneTimeAccessData;
}

export const OneTimeAccessEmail = ({ data, ...props }: OneTimeAccessEmailProps) => (
	<BaseTemplate {...props} preview={`Your ${props.appName} login code is ${data.code}`}>
		<CardHeader title="Your login code">
			Enter this code to sign in to {props.appName}. It expires in {data.expirationString}.
		</CardHeader>

		<CodeBox code={data.code} />

		<Button href={data.buttonCodeLink}>Sign in</Button>

		<Muted>
			Or open <MutedLink href={data.loginLink}>{data.loginLink}</MutedLink> and enter the code
			manually. If you didn't ask for this code, you can ignore this email.
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
		expirationString: '{{.Data.ExpirationString}}'
	}
};

OneTimeAccessEmail.PreviewProps = {
	...sharedPreviewProps,
	data: {
		code: '123456',
		loginLink: 'https://id.example.com/lc',
		buttonCodeLink: 'https://id.example.com/lc/123456',
		expirationString: '15 minutes'
	}
};
