import { BaseTemplate } from "../components/base-template";
import CardHeader from "../components/card-header";
import { DetailsList } from "../components/details-list";
import { Muted, Paragraph } from "../components/text";
import {
  type SharedProps,
  sharedPreviewProps,
  sharedTemplateProps,
} from "../props";

interface SignInData {
  location: string;
  ipAddress: string;
  device: string;
  dateTime: string;
}

interface NewSignInEmailProps extends SharedProps {
  data: SignInData;
}

export const NewSignInEmail = ({ data, ...props }: NewSignInEmailProps) => (
  <BaseTemplate
    {...props}
    preview={`A new sign-in to your ${props.appName} account was detected`}
  >
    <CardHeader title="New sign-in detected" />
    <Paragraph>
      Your {props.appName} account was recently accessed from a new IP address
      or browser. If this was you, no further action is needed.
    </Paragraph>

    <DetailsList
      items={[
        { label: "Approximate location", value: data.location },
        { label: "IP address", value: data.ipAddress },
        { label: "Device", value: data.device },
        { label: "Time", value: data.dateTime },
      ]}
    />

    <Muted>
      If you don't recognize this activity, review the passkeys in your{" "}
      {props.appName} account settings and remove any you don't recognize.
    </Muted>
  </BaseTemplate>
);

export default NewSignInEmail;

NewSignInEmail.TemplateProps = {
  ...sharedTemplateProps,
  data: {
    location:
      "{{if and .Data.City .Data.Country}}{{.Data.City}}, {{.Data.Country}}{{else if .Data.Country}}{{.Data.Country}}{{else}}Unknown{{end}}",
    ipAddress: "{{.Data.IPAddress}}",
    device: "{{.Data.Device}}",
    dateTime: '{{.Data.DateTime.Format "January 2, 2006 at 3:04 PM MST"}}',
  },
};

NewSignInEmail.PreviewProps = {
  ...sharedPreviewProps,
  data: {
    location: "San Francisco, USA",
    ipAddress: "203.0.113.42",
    device: "Chrome on macOS",
    dateTime: "January 2, 2026 at 3:04 PM UTC",
  },
};
