import {
  Body,
  Column,
  Container,
  Head,
  Html,
  Img,
  Row,
  Section,
  Text,
} from "@react-email/components";

interface BaseTemplateProps {
  logoURL?: string;
  appName: string;
  children: React.ReactNode;
}

export const BaseTemplate = ({
  logoURL,
  appName,
  children,
}: BaseTemplateProps) => {
  const finalLogoURL =
    logoURL ||
    "https://private-user-images.githubusercontent.com/58886915/359183039-4ceb2708-9f29-4694-b797-be833efce17d.png?jwt=eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJnaXRodWIuY29tIiwiYXVkIjoicmF3LmdpdGh1YnVzZXJjb250ZW50LmNvbSIsImtleSI6ImtleTUiLCJleHAiOjE3NTY0NTk5MzksIm5iZiI6MTc1NjQ1OTYzOSwicGF0aCI6Ii81ODg4NjkxNS8zNTkxODMwMzktNGNlYjI3MDgtOWYyOS00Njk0LWI3OTctYmU4MzNlZmNlMTdkLnBuZz9YLUFtei1BbGdvcml0aG09QVdTNC1ITUFDLVNIQTI1NiZYLUFtei1DcmVkZW50aWFsPUFLSUFWQ09EWUxTQTUzUFFLNFpBJTJGMjAyNTA4MjklMkZ1cy1lYXN0LTElMkZzMyUyRmF3czRfcmVxdWVzdCZYLUFtei1EYXRlPTIwMjUwODI5VDA5MjcxOVomWC1BbXotRXhwaXJlcz0zMDAmWC1BbXotU2lnbmF0dXJlPWM4ZWI5NzlkMDA5NDNmZGU5MjQwMGE1YjA0NWZiNzEzM2E0MzAzOTFmOWRmNDUzNmJmNjQwZTMxNGIzZmMyYmQmWC1BbXotU2lnbmVkSGVhZGVycz1ob3N0In0.YdfLv1tD5KYnRZPSA3QlR1SsvScpP0rt-J3YD6ZHsCk";

  return (
    <Html>
      <Head />
      <Body style={mainStyle}>
        <Container style={{ width: "500px", margin: "0 auto" }}>
          <Section>
            <Row
            align="left"
            style={{
              width: "210px",
              marginBottom: "16px",
            }}
          >
            <Column>
              <Img
                src={finalLogoURL}
                width="32"
                height="32"
                alt={appName}
                style={logoStyle}
              />
            </Column>
            <Column>
              <Text style={titleStyle}>{appName}</Text>
            </Column>
          </Row>
          </Section>
          <div style={content}>{children}</div>
        </Container>
      </Body>
    </Html>
  );
};

const mainStyle = {
  padding: "50px",
  backgroundColor: "#FBFBFB",
  fontFamily: "Arial, sans-serif",
};

const logoStyle = {
  width: "32px",
  height: "32px",
  verticalAlign: "middle",
  marginRight: "8px",
};

const titleStyle = {
  fontSize: "23px",
  fontWeight: "bold",
  margin: "0",
  padding: "0",
};

const content = {
  backgroundColor: "white",
  padding: "24px",
  borderRadius: "10px",
  boxShadow: "0 1px 4px 0px rgba(0, 0, 0, 0.1)",
};
