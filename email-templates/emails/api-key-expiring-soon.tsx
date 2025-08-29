import { Text } from "@react-email/components";
import { BaseTemplate } from "../components/base-template";
import CardHeader from "../components/card-header";
import { sharedPreviewProps, sharedTemplateProps } from "../props";

interface ApiKeyExpiringData {
  name: string;
  apiKeyName: string;
  expiresAt: string;
}

interface ApiKeyExpiringEmailProps {
  logoURL: string;
  appName: string;
  data: ApiKeyExpiringData;
}

export const ApiKeyExpiringEmail = ({
  logoURL,
  appName,
  data,
}: ApiKeyExpiringEmailProps) => (
  <BaseTemplate logoURL={logoURL} appName={appName}>
    <CardHeader title="API Key Expiring Soon" warning />
    <Text>
      Hello {data.name}, <br />
      This is a reminder that your API key <strong>
        {data.apiKeyName}
      </strong>{" "}
      will expire on <strong>{data.expiresAt}</strong>.
    </Text>

    <Text>Please generate a new API key if you need continued access.</Text>
  </BaseTemplate>
);

export default ApiKeyExpiringEmail;

ApiKeyExpiringEmail.TemplateProps = {
  ...sharedTemplateProps,
  data: {
    name: "{{.Data.Name}}",
    apiKeyName: "{{.Data.APIKeyName}}",
    expiresAt: '{{.Data.ExpiresAt.Format "2006-01-02 15:04:05 MST"}}',
  },
};

ApiKeyExpiringEmail.PreviewProps = {
  ...sharedPreviewProps,
  data: {
    name: "Elias Schneider",
    apiKeyName: "My API Key",
    expiresAt: "September 30, 2024",
  },
};
