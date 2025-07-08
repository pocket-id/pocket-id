import { Body, Container, Head, Html, Img, Section, Text } from '@react-email/components';

interface BaseTemplateProps {
  logoURL?: string; // Make logoURL optional
  appName: string;
  children: React.ReactNode;
}

export const BaseTemplate = ({ logoURL, appName, children }: BaseTemplateProps) => {
  // Fallback to local pocketid.png if no logoURL provided
  const finalLogoURL = logoURL || '/static/pocketid.png';

  return (
    <Html lang="en">
      <Head />
      <Body style={body}>
        <Container style={container}>
          <Section style={header}>
            <div style={logo}>
              <Img src={finalLogoURL} alt={appName} width="32" height="32" style={logoImg} />
              <Text style={appTitle}>{appName}</Text>
            </div>
          </Section>
          <Section style={content}>{children}</Section>
        </Container>
      </Body>
    </Html>
  );
};

const body = {
  margin: 0,
  padding: '0 16px',
  backgroundColor: '#f0f0f0',
  color: '#333',
  fontFamily: 'Arial, sans-serif',
  lineHeight: '1.5',
};

const container = {
  width: '100%',
  maxWidth: '600px',
  margin: '40px auto',
  backgroundColor: '#fff',
  borderRadius: '10px',
  boxShadow: '0 4px 6px rgba(0, 0, 0, 0.1)',
  padding: '32px',
};

const header = {
  display: 'flex',
  marginBottom: '24px',
};

const logo = {
  display: 'flex',
  alignItems: 'center',
};

const logoImg = {
  verticalAlign: 'middle',
};

const appTitle = {
  fontSize: '1.5rem',
  fontWeight: 'bold',
  display: 'inline-block',
  verticalAlign: 'middle',
  marginLeft: '8px',
  margin: 0,
};

const content = {
  backgroundColor: '#fafafa',
  padding: '24px',
  borderRadius: '10px',
};
