import { Link, Text } from "@react-email/components";
import { BaseTemplate } from "../components/base-template";
import { Button } from "../components/button";
import CardHeader from "../components/card-header";
import { sharedPreviewProps, sharedTemplateProps } from "../props";

interface OneTimeAccessData {
  code: string;
  loginLink: string;
  buttonCodeLink: string;
  expirationString: string;
}

interface OneTimeAccessEmailProps {
  logoURL: string;
  appName: string;
  data: OneTimeAccessData;
}

export const OneTimeAccessEmail = ({
  logoURL,
  appName,
  data,
}: OneTimeAccessEmailProps) => (
  <BaseTemplate logoURL={logoURL} appName={appName}>
    <CardHeader title="Your Login Code" />

    <Text>
      Click the button below to sign in to {appName} with a login code.
      <br />
      Or visit{" "}
      <Link href={data.loginLink} style={linkStyle}>
        {data.loginLink}
      </Link>{" "}
      and enter the code <strong>{data.code}</strong>.
      <br />
      <br />
      This code expires in {data.expirationString}.
    </Text>

    <Button href={data.buttonCodeLink}>Sign In</Button>
  </BaseTemplate>
);

export default OneTimeAccessEmail;

const linkStyle = {
  color: "#000",
  textDecoration: "underline",
  fontFamily: "Arial, sans-serif",
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
    loginLink: "https://example.com/login",
    buttonCodeLink: "https://example.com/login?code=123456",
    expirationString: "15 minutes",
  },
};
