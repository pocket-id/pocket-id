import { BaseTemplate } from "../components/base-template";
import CardHeader from "../components/card-header";
import { Muted, Paragraph } from "../components/text";
import {
  type SharedProps,
  sharedPreviewProps,
  sharedTemplateProps,
} from "../props";

interface ApiKeyExpiringData {
  name: string;
  apiKeyName: string;
  expiresAt: string;
}

interface ApiKeyExpiringEmailProps extends SharedProps {
  data: ApiKeyExpiringData;
}

export const ApiKeyExpiringEmail = ({
  data,
  ...props
}: ApiKeyExpiringEmailProps) => (
  <BaseTemplate
    {...props}
    preview={`Your API key ${data.apiKeyName} expires on ${data.expiresAt}`}
  >
    <CardHeader title="API key expiring soon" />
    <Paragraph>Hello {data.name},</Paragraph>
    <Paragraph>
      Your API key <strong>{data.apiKeyName}</strong> will expire on{" "}
      <strong>{data.expiresAt}</strong>. Anything that uses this key will stop
      working once it expires.
    </Paragraph>

    <Muted>
      To keep access, create a new API key in your {props.appName} account
      settings before then.
    </Muted>
  </BaseTemplate>
);

export default ApiKeyExpiringEmail;

ApiKeyExpiringEmail.TemplateProps = {
  ...sharedTemplateProps,
  data: {
    name: "{{.Data.Name}}",
    apiKeyName: "{{.Data.ApiKeyName}}",
    expiresAt: '{{.Data.ExpiresAt.Format "2006-01-02 15:04:05 MST"}}',
  },
};

ApiKeyExpiringEmail.PreviewProps = {
  ...sharedPreviewProps,
  data: {
    name: "Elias",
    apiKeyName: "CI deploy key",
    expiresAt: "2026-01-30 12:00:00 UTC",
  },
};
