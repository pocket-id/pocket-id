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
  return (
    <Html>
      <Head />
      <Body style={mainStyle}>
        <Container style={{ width: "500px", margin: "0 auto" }}>
          <Section>
            <Row
              align="left"
              style={{
                marginBottom: "16px",
              }}
            >
              <Column style={{ width: "50px" }}>
                <Img
                  src={logoURL}
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
