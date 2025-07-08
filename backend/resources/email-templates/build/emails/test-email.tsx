import { Text } from '@react-email/components';
import { BaseTemplate } from './base-template';

interface TestEmailProps {
  logoURL: string;
  appName: string;
}

export const TestEmail = ({ logoURL, appName }: TestEmailProps) => (
  <BaseTemplate logoURL={logoURL} appName={appName}>
    <Text style={paragraph}>This is a test email.</Text>
  </BaseTemplate>
);

export default TestEmail;

const paragraph = {
  fontSize: '1rem',
  lineHeight: '1.5',
  margin: '0',
  padding: '0',
  color: '#333',
  fontFamily: '-apple-system,BlinkMacSystemFont,"Segoe UI",Helvetica,Arial,sans-serif,"Apple Color Emoji","Segoe UI Emoji"',
};
