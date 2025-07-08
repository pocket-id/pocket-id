import { Body, Container, Head, Html, Img, Text } from '@react-email/components';

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
          <div style={header}>
            <Img
              src={finalLogoURL}
              width="32"
              height="32"
              alt={appName}
              style={logo}
            />
            <Text style={title}>
              <strong>{appName}</strong>
            </Text>
          </div>

          <div style={section}>
            {children}
          </div>
        </Container>
      </Body>
    </Html>
  );
};

const main = {
  backgroundColor: '#ffffff',
  color: '#24292e',
  fontFamily: '-apple-system,BlinkMacSystemFont,"Segoe UI",Helvetica,Arial,sans-serif,"Apple Color Emoji","Segoe UI Emoji"',
  margin: '0',
  padding: '0',
};

const container = {
  maxWidth: '480px',
  margin: '0 auto',
  padding: '20px 0 48px',
};

const header = {
  display: 'flex',
  alignItems: 'center',
  marginBottom: '24px',
};

const logo = {
  marginRight: '12px',
  display: 'block',
};

const title = {
  fontSize: '24px',
  lineHeight: 1.25,
  margin: '0',
  padding: '0',
  fontFamily: '-apple-system,BlinkMacSystemFont,"Segoe UI",Helvetica,Arial,sans-serif,"Apple Color Emoji","Segoe UI Emoji"',
};

const section = {
  padding: '24px',
  border: 'solid 1px #dedede',
  borderRadius: '5px',
  textAlign: 'left' as const,
  backgroundColor: '#ffffff',
  margin: '0',
  boxSizing: 'border-box' as const,
};
