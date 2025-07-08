import { Text } from '@react-email/components';
import { BaseTemplate } from './base-template';
import { WarningBadge } from './components/warning-badge';

interface SignInData {
  city?: string;
  country?: string;
  ipAddress: string;
  device: string;
  dateTime: string;
}

interface NewSignInEmailProps {
  logoURL: string;
  appName: string;
  data: SignInData;
}

export const NewSignInEmail = ({ logoURL, appName, data }: NewSignInEmailProps) => (
  <BaseTemplate logoURL={logoURL} appName={appName}>
    <div style={headerSection}>
      <Text style={title}>New Sign-In Detected</Text>
      <WarningBadge>Warning</WarningBadge>
    </div>

    <div style={gridSection}>
      <div style={gridRow}>
        {data.city && data.country && (
          <div style={gridCell}>
            <Text style={label}>Approximate Location</Text>
            <Text style={value}>
              {data.city}, {data.country}
            </Text>
          </div>
        )}
        <div style={gridCell}>
          <Text style={label}>IP Address</Text>
          <Text style={value}>{data.ipAddress}</Text>
        </div>
      </div>
      <div style={gridRow}>
        <div style={gridCell}>
          <Text style={label}>Device</Text>
          <Text style={value}>{data.device}</Text>
        </div>
        <div style={gridCell}>
          <Text style={label}>Sign-In Time</Text>
          <Text style={value}>{data.dateTime}</Text>
        </div>
      </div>
    </div>

    <Text style={message}>
      This sign-in was detected from a new device or location. If you recognize this activity, you can safely ignore this message. If not, please review your account and security settings.
    </Text>
  </BaseTemplate>
);

export default NewSignInEmail;

const headerSection = {
  display: 'flex',
  justifyContent: 'space-between',
  alignItems: 'center',
  marginBottom: '24px',
};

const title = {
  fontSize: '1.5rem',
  fontWeight: 'bold' as const,
  margin: '0',
  padding: '0',
  color: '#333',
  fontFamily: '-apple-system,BlinkMacSystemFont,"Segoe UI",Helvetica,Arial,sans-serif,"Apple Color Emoji","Segoe UI Emoji"',
};

const gridSection = {
  marginBottom: '24px',
};

const gridRow = {
  display: 'flex',
  marginBottom: '12px',
};

const gridCell = {
  flex: '1',
  padding: '12px 16px',
  border: 'none',
  margin: '0',
  backgroundColor: 'transparent',
  fontFamily: '-apple-system,BlinkMacSystemFont,"Segoe UI",Helvetica,Arial,sans-serif,"Apple Color Emoji","Segoe UI Emoji"',
};

const label = {
  fontSize: '0.875rem',
  fontWeight: 'bold' as const,
  color: '#666',
  margin: '0 0 4px 0',
  padding: '0',
  display: 'block',
  fontFamily: '-apple-system,BlinkMacSystemFont,"Segoe UI",Helvetica,Arial,sans-serif,"Apple Color Emoji","Segoe UI Emoji"',
};

const value = {
  fontSize: '1rem',
  color: '#333',
  margin: '0',
  padding: '0',
  display: 'block',
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
