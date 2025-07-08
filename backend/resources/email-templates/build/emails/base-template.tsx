import { Body, Container, Head, Html, Img, Section, Text } from '@react-email/components';

interface BaseTemplateProps {
  logoURL?: string;
  appName: string;
  children: React.ReactNode;
}

export const BaseTemplate = ({ logoURL, appName, children }: BaseTemplateProps) => {
  // Fallback to local pocketid.png if no logoURL provided
  const finalLogoURL = logoURL || '/static/pocketid.png';

  return (
    <Html>
      <Head />
      <Body style={main}>
        <Container style={container}>
          <Img
            src={finalLogoURL}
            width="32"
            height="32"
            alt={appName}
          />

          <Text style={title}>
            <strong>{appName}</strong>
          </Text>

          <Section style={section}>
            {children}
          </Section>
        </Container>
      </Body>
    </Html>
  );
};

const main = {
  backgroundColor: '#ffffff',
  color: '#24292e',
  fontFamily:
    '-apple-system,BlinkMacSystemFont,"Segoe UI",Helvetica,Arial,sans-serif,"Apple Color Emoji","Segoe UI Emoji"',
};

const container = {
  maxWidth: '480px',
  margin: '0 auto',
  padding: '20px 0 48px',
};

const title = {
  fontSize: '24px',
  lineHeight: 1.25,
  margin: '16px 0',
};

const section = {
  padding: '24px',
  border: 'solid 1px #dedede',
  borderRadius: '5px',
  textAlign: 'center' as const,
};
