import { Text } from '@react-email/components';
import { BaseTemplate } from './base-template';
import { Button } from './components/button';

interface OneTimeAccessData {
  code: string;
  loginLink: string;
  buttonCodeLink: string;
  expirationString: string;
}

interface OneTimeAccessEmailProps {
  logoURL: string;
  appName: string;
  data: OneTimeAccessData;
}

export const OneTimeAccessEmail = ({ logoURL, appName, data }: OneTimeAccessEmailProps) => (
  <BaseTemplate logoURL={logoURL} appName={appName}>
    <Text style={contentTitle}>Login Code</Text>

    <Text style={message}>
      Click the button below to sign in to {appName} with a login code.
      <br />
      Or visit <a href={data.loginLink} style={linkStyle}>{data.loginLink}</a> and enter the code <strong>{data.code}</strong>.
      <br />
      <br />
      This code expires in {data.expirationString}.
    </Text>

    <Button href={data.buttonCodeLink}>Sign In</Button>
  </BaseTemplate>
);

export default OneTimeAccessEmail;

const contentTitle = {
  fontSize: '1.25rem',
  fontWeight: 'bold' as const,
  marginBottom: '16px',
  margin: '0 0 16px 0',
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

const linkStyle = {
  color: '#000',
  textDecoration: 'underline',
  fontFamily: 'Arial, sans-serif',
};