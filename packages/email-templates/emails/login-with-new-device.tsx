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
	<BaseTemplate {...props} preview={props.t('new_sign_in_preview', { appName: props.appName })}>
		<CardHeader title={props.t('new_sign_in_title')}>
			{props.t('new_sign_in_description', { appName: props.appName })}
		</CardHeader>

		<DetailsList
			items={[
				{ label: props.t('new_sign_in_location'), value: data.location },
				{ label: props.t('new_sign_in_ip_address'), value: data.ipAddress },
				{ label: props.t('new_sign_in_device'), value: data.device },
				{ label: props.t('new_sign_in_method'), value: data.method },
				{ label: props.t('new_sign_in_time'), value: data.dateTime }
			]}
		/>

		<Muted>
			{props.t('new_sign_in_review', {}, 0)}
			<MutedLink href={`${props.appURL}/settings/account`}>
				{props.t('new_sign_in_review', {}, 1)}
			</MutedLink>
			{props.t('new_sign_in_review', {}, 2)}
		</Muted>
	</BaseTemplate>
);

export default NewSignInEmail;

NewSignInEmail.TemplateProps = {
	...sharedTemplateProps,
	data: {
		location:
			'{{if and .Data.City .Data.Country}}{{.Data.City}}, {{.Data.Country}}{{else if .Data.Country}}{{.Data.Country}}{{else}}{{.T "unknown"}}{{end}}',
		ipAddress: '{{.Data.IPAddress}}',
		device: '{{.Data.Device}}',
		method: '{{.T .Data.Method}}',
		dateTime: '{{.DateTime .Data.DateTime}}'
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
