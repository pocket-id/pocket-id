import { Text, Section } from '@react-email/components';
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
    <Section style={headerSection}>
      <Text style={title}>API Key Expiring Soon</Text>
      <WarningBadge>Warning</WarningBadge>
    </Section>

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

const title = {
  fontSize: '1.25rem',
  fontWeight: 'bold' as const,
  margin: '0',
  padding: '0',
  color: '#333',
  fontFamily: '-apple-system,BlinkMacSystemFont,"Segoe UI",Helvetica,Arial,sans-serif,"Apple Color Emoji","Segoe UI Emoji"',
};

const message = {
  fontSize: '1rem',
  lineHeight: '1.6',
  color: '#333',
  margin: '0',
  padding: '0',
  fontFamily: '-apple-system,BlinkMacSystemFont,"Segoe UI",Helvetica,Arial,sans-serif,"Apple Color Emoji","Segoe UI Emoji"',
};