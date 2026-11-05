import { BaseTemplate } from '../components/base-template';
import { Button } from '../components/button';
import CardHeader from '../components/card-header';
import { DetailsList } from '../components/details-list';
import { type SharedProps, sharedPreviewProps, sharedTemplateProps } from '../props';

interface ApiKeyExpiringData {
	name: string;
	apiKeyName: string;
	expiresAt: string;
}

interface ApiKeyExpiringEmailProps extends SharedProps {
	data: ApiKeyExpiringData;
}

export const ApiKeyExpiringEmail = ({ data, ...props }: ApiKeyExpiringEmailProps) => (
	<BaseTemplate {...props} preview={`Your API key ${data.apiKeyName} expires on ${data.expiresAt}`}>
		<CardHeader title="API key expiring soon">
			Hello {data.name}, one of your API keys expires soon. Anything that uses it will stop working
			once it expires.
		</CardHeader>

		<DetailsList
			items={[
				{ label: 'API key', value: data.apiKeyName },
				{ label: 'Expires', value: data.expiresAt }
			]}
		/>

		<Button href={`${props.appURL}/settings/account`}>Renew API key</Button>
	</BaseTemplate>
);

export default ApiKeyExpiringEmail;

ApiKeyExpiringEmail.TemplateProps = {
	...sharedTemplateProps,
	data: {
		name: '{{.Data.Name}}',
		apiKeyName: '{{.Data.ApiKeyName}}',
		expiresAt: '{{.Data.ExpiresAt.Format "2006-01-02 15:04:05 MST"}}'
	}
};

ApiKeyExpiringEmail.PreviewProps = {
	...sharedPreviewProps,
	data: {
		name: 'Elias',
		apiKeyName: 'CI deploy key',
		expiresAt: '2026-01-30 12:00:00 UTC'
	}
};
