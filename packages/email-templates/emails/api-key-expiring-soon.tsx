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
	<BaseTemplate
		{...props}
		preview={props.t('api_key_expiring_preview', {
			apiKeyName: data.apiKeyName,
			expiresAt: data.expiresAt
		})}
	>
		<CardHeader title={props.t('api_key_expiring_title')}>
			{props.t('api_key_expiring_description', { name: data.name })}
		</CardHeader>

		<DetailsList
			items={[
				{ label: props.t('api_key_expiring_name'), value: data.apiKeyName },
				{ label: props.t('api_key_expiring_expires'), value: data.expiresAt }
			]}
		/>

		<Button href={`${props.appURL}/settings/account`}>{props.t('api_key_expiring_button')}</Button>
	</BaseTemplate>
);

export default ApiKeyExpiringEmail;

ApiKeyExpiringEmail.TemplateProps = {
	...sharedTemplateProps,
	data: {
		name: '{{.Data.Name}}',
		apiKeyName: '{{.Data.ApiKeyName}}',
		expiresAt: '{{.DateTime .Data.ExpiresAt}}'
	}
};

ApiKeyExpiringEmail.PreviewProps = {
	...sharedPreviewProps,
	data: {
		name: 'Elias',
		apiKeyName: 'CI deploy key',
		expiresAt: 'January 30, 2026 at 12:00 PM UTC'
	}
};
