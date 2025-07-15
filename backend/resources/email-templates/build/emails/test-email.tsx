import { Text } from '@react-email/components';
import { BaseTemplate } from './base-template';

interface TestEmailProps {
  logoURL: string;
  appName: string;
}

export const TestEmail = ({ logoURL, appName }: TestEmailProps) => (
  <BaseTemplate logoURL={logoURL} appName={appName}>
    <Text style={message}>This is a test email.</Text>
  </BaseTemplate>
);

export default TestEmail;

const message = {
  fontSize: '1rem',
  lineHeight: '1.5',
  marginTop: '16px',
  margin: '0',
  padding: '0',
  color: '#333',
  fontFamily: 'Arial, sans-serif',
};
