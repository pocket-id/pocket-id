import { BaseTemplate } from '../components/base-template';
import CardHeader from '../components/card-header';
import { DetailsList } from '../components/details-list';
import { Muted, MutedLink } from '../components/text';
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
}

export const NewSignInEmail = ({ data, ...props }: NewSignInEmailProps) => (
	<BaseTemplate {...props} preview={`A new sign-in to your ${props.appName} account was detected`}>
		<CardHeader title="New sign-in detected">
			Your {props.appName} account was accessed from a new IP address or browser. If this was you,
			there's nothing to do.
		</CardHeader>

		<DetailsList
			items={[
				{ label: 'Approximate location', value: data.location },
				{ label: 'IP address', value: data.ipAddress },
				{ label: 'Device', value: data.device },
				{ label: 'Sign-in method', value: data.method },
				{ label: 'Time', value: data.dateTime }
			]}
		/>

		<Muted>
			Don't recognize this? Review the passkeys in your{' '}
			<MutedLink href={`${props.appURL}/settings/account`}>account settings</MutedLink> and remove
			any you don't know.
		</Muted>
	</BaseTemplate>
);

export default NewSignInEmail;

NewSignInEmail.TemplateProps = {
	...sharedTemplateProps,
	data: {
		location:
			'{{if and .Data.City .Data.Country}}{{.Data.City}}, {{.Data.Country}}{{else if .Data.Country}}{{.Data.Country}}{{else}}Unknown{{end}}',
		ipAddress: '{{.Data.IPAddress}}',
		device: '{{.Data.Device}}',
		method: '{{.Data.Method}}',
		dateTime: '{{.Data.DateTime.Format "January 2, 2006 at 3:04 PM MST"}}'
	}
};

NewSignInEmail.PreviewProps = {
	...sharedPreviewProps,
	data: {
		location: 'San Francisco, USA',
		ipAddress: '203.0.113.42',
		device: 'Chrome on macOS',
		method: 'Passkey',
		dateTime: 'January 2, 2026 at 3:04 PM UTC'
	}
};
