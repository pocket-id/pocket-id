import { Column, Heading, Row, Text } from "@react-email/components";
import { BaseTemplate } from "../components/base-template";
import CardHeader from "../components/card-header";
import { sharedPreviewProps, sharedTemplateProps } from "../props";

interface SignInData {
  location: string;
  ipAddress: string;
  device: string;
  dateTime: string;
}

interface NewSignInEmailProps {
  logoURL: string;
  appName: string;
  data: SignInData;
}

export const NewSignInEmail = ({
  logoURL,
  appName,
  data,
}: NewSignInEmailProps) => (
  <BaseTemplate logoURL={logoURL} appName={appName}>
    <CardHeader title="New Sign-In Detected" warning />
    <Text>
      Your {appName} account was recently accessed from a new IP address or
      browser. If you recognize this activity, no further action is required.
    </Text>
    <Heading
      style={{
        fontSize: "1rem",
        fontWeight: "bold",
        margin: "30px 0 10px 0",
      }}
      as="h4"
    >
      Details
    </Heading>

    <Row>
      <Column style={detailsBoxStyle}>
        <Text style={detailsLabelStyle}>Approximate Location</Text>
        <Text style={detailsBoxValueStyle}>{data.location}</Text>
      </Column>
      <Column style={detailsBoxStyle}>
        <Text style={detailsLabelStyle}>IP Address</Text>
        <Text style={detailsBoxValueStyle}>{data.ipAddress}</Text>
      </Column>
    </Row>

    <Row style={{ marginTop: "10px" }}>
      <Column style={detailsBoxStyle}>
        <Text style={detailsLabelStyle}>Device</Text>
        <Text style={detailsBoxValueStyle}>{data.device}</Text>
      </Column>
      <Column style={detailsBoxStyle}>
        <Text style={detailsLabelStyle}>Sign-In Time</Text>
        <Text style={detailsBoxValueStyle}>{data.dateTime}</Text>
      </Column>
    </Row>
  </BaseTemplate>
);

export default NewSignInEmail;

const detailsBoxStyle = {
  width: "225px",
};

const detailsLabelStyle = {
  margin: 0,
  fontSize: "12px",
  color: "gray",
};

const detailsBoxValueStyle = {
  margin: 0,
};

NewSignInEmail.TemplateProps = {
  ...sharedTemplateProps,
  data: {
    location: "{{if and .Data.City .Data.Country}}{{.Data.City}}, {{.Data.Country}}{{else if .Data.Country}}{{.Data.Country}}{{else}}Unknown{{end}}",
    ipAddress: "{{.Data.IPAddress}}",
    device: "{{.Data.Device}}",
    dateTime: '{{.Data.DateTime.Format "January 2, 2006 at 3:04 PM MST"}}',
  },
};

NewSignInEmail.PreviewProps = {
  ...sharedPreviewProps,
  data: {
    location: "San Francisco, USA",
    ipAddress: "127.0.0.1",
    device: "Chrome on macOS",
    dateTime: "2024-01-01 12:00 PM UTC",
  },
};
