import { Text } from "@react-email/components";
import { BaseTemplate } from "../components/base-template";
import { Button } from "../components/button";
import CardHeader from "../components/card-header";
import { sharedPreviewProps, sharedTemplateProps } from "../props";

interface EmailVerificationData {
  userFullName: string;
  verificationLink: string;
}

interface EmailVerificationProps {
  logoURL: string;
  appName: string;
  data: EmailVerificationData;
}

export const EmailVerification = ({
  logoURL,
  appName,
  data,
}: EmailVerificationProps) => (
  <BaseTemplate logoURL={logoURL} appName={appName}>
    <CardHeader title="Email Verification" />

    <Text>
      Hello {data.userFullName}, <br />
      Click the button below to verify your email address for {appName}. This
      link will expire in 24 hours.
      <br />
    </Text>

    <Button href={data.verificationLink}>Verify</Button>
  </BaseTemplate>
);

export default EmailVerification;

EmailVerification.TemplateProps = {
  ...sharedTemplateProps,
  data: {
    userFullName: "{{.Data.UserFullName}}",
    verificationLink: "{{.Data.VerificationLink}}",
  },
};

EmailVerification.PreviewProps = {
  ...sharedPreviewProps,
  data: {
    userFullName: "Tim Cook",
    verificationLink:
      "https://localhost:1411/user/verify-email?code=abcdefg12345",
  },
};
