import { Body, Head, Html, Img, Text } from '@react-email/components';

interface BaseTemplateProps {
  logoURL?: string;
  appName: string;
  children: React.ReactNode;
}

export const BaseTemplate = ({ logoURL, appName, children }: BaseTemplateProps) => {
  const finalLogoURL = logoURL || '/static/pocketid.png';

  return (
    <Html>
      <Head />
      <Body style={main}>
        <div style={container}>
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

          <div style={content}>
            {children}
          </div>
        </div>
      </Body>
    </Html>
  );
};

const main = {
  margin: '0',
  padding: '16px',
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
  width: '32px',
  height: '32px',
  verticalAlign: 'middle',
  marginRight: '8px',
};

const title = {
  fontSize: '1.5rem',
  fontWeight: 'bold',
  display: 'inline-block',
  verticalAlign: 'middle',
  margin: '0',
  padding: '0',
};

const content = {
  backgroundColor: '#fafafa',
  padding: '24px',
  borderRadius: '10px',
};
