import { Text } from '@react-email/components';
import { BaseTemplate } from './base-template';
import { Button } from './components/button';

interface OneTimeAccessData {
  code: string;
  loginLink: string;
  loginLinkWithCode: string;
  expirationString: string;
}

interface OneTimeAccessEmailProps {
  logoURL: string;
  appName: string;
  data: OneTimeAccessData;
}

export const OneTimeAccessEmail = ({ logoURL, appName, data }: OneTimeAccessEmailProps) => (
  <BaseTemplate logoURL={logoURL} appName={appName}>
    <div style={headerSection}>
      <Text style={title}>Login Code</Text>
    </div>

    <Text style={message}>
      Click the button below to sign in to {appName} with a login code.
      <br />
      Or visit <a href={data.loginLink} style={linkStyle}>{data.loginLink}</a> and enter the code <strong>{data.code}</strong>.
      <br />
      <br />
      This code expires in {data.expirationString}.
    </Text>

    <Button href={data.loginLinkWithCode}>Sign In</Button>
  </BaseTemplate>
);

export default OneTimeAccessEmail;

const headerSection = {
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

const linkStyle = {
  color: '#000',
  textDecoration: 'underline',
  fontFamily: '-apple-system,BlinkMacSystemFont,"Segoe UI",Helvetica,Arial,sans-serif,"Apple Color Emoji","Segoe UI Emoji"',
};