import { Text } from '@react-email/components';
import { BaseTemplate } from './base-template';
import { WarningBadge } from './components/warning-badge';

interface ApiKeyExpiringData {
  name: string;
  apiKeyName: string;
  expiresAt: string;
}

interface ApiKeyExpiringEmailProps {
  logoURL: string;
  appName: string;
  data: ApiKeyExpiringData;
}

export const ApiKeyExpiringEmail = ({ logoURL, appName, data }: ApiKeyExpiringEmailProps) => (
  <BaseTemplate logoURL={logoURL} appName={appName}>
    <div style={headerSection}>
      <Text style={contentTitle}>API Key Expiring Soon</Text>
      <WarningBadge>Warning</WarningBadge>
    </div>

    <Text style={message}>
      Hello {data.name},
      <br />
      <br />
      This is a reminder that your API key <strong>{data.apiKeyName}</strong> will expire on <strong>{data.expiresAt}</strong>.
      <br />
      <br />
      Please generate a new API key if you need continued access.
    </Text>
  </BaseTemplate>
);

export default ApiKeyExpiringEmail;

const headerSection = {
  display: 'flex',
  justifyContent: 'space-between',
  alignItems: 'center',
  marginBottom: '24px',
};

const contentTitle = {
  fontSize: '1.25rem',
  fontWeight: 'bold' as const,
  marginBottom: '16px',
  margin: '0',
  padding: '0',
  color: '#333',
  fontFamily: 'Arial, sans-serif',
};

const message = {
  fontSize: '1rem',
  lineHeight: '1.5',
  marginTop: '16px',
  margin: '0',
  padding: '0',
  color: '#333',
  fontFamily: 'Arial, sans-serif',
};