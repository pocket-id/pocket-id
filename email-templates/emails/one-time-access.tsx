import { Link } from "react-email";
import { BaseTemplate } from "../components/base-template";
import { Button } from "../components/button";
import CardHeader from "../components/card-header";
import { CodeBox } from "../components/code-box";
import { Muted, Paragraph } from "../components/text";
import { colors } from "../components/theme";
import {
  type SharedProps,
  sharedPreviewProps,
  sharedTemplateProps,
} from "../props";

interface OneTimeAccessData {
  code: string;
  loginLink: string;
  buttonCodeLink: string;
  expirationString: string;
}

interface OneTimeAccessEmailProps extends SharedProps {
  data: OneTimeAccessData;
}

export const OneTimeAccessEmail = ({
  data,
  ...props
}: OneTimeAccessEmailProps) => (
  <BaseTemplate
    {...props}
    preview={`Your ${props.appName} login code is ${data.code}`}
  >
    <CardHeader title="Your login code" />
    <Paragraph>
      Use the code below to sign in to {props.appName}. It expires in{" "}
      {data.expirationString}.
    </Paragraph>

    <CodeBox code={data.code} />

    <Button href={data.buttonCodeLink}>Sign in</Button>

    <Muted style={{ marginTop: "32px" }}>
      Or open{" "}
      <Link href={data.loginLink} style={linkStyle}>
        {data.loginLink}
      </Link>{" "}
      and enter the code manually.
    </Muted>
  </BaseTemplate>
);

export default OneTimeAccessEmail;

const linkStyle = {
  color: colors.mutedForeground,
  textDecoration: "underline",
  wordBreak: "break-all" as const,
};

OneTimeAccessEmail.TemplateProps = {
  ...sharedTemplateProps,
  data: {
    code: "{{.Data.Code}}",
    loginLink: "{{.Data.LoginLink}}",
    buttonCodeLink: "{{.Data.LoginLinkWithCode}}",
    expirationString: "{{.Data.ExpirationString}}",
  },
};

OneTimeAccessEmail.PreviewProps = {
  ...sharedPreviewProps,
  data: {
    code: "123456",
    loginLink: "https://id.example.com/lc",
    buttonCodeLink: "https://id.example.com/lc/123456",
    expirationString: "15 minutes",
  },
};
