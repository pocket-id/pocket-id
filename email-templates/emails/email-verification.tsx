import { Link } from "react-email";
import { BaseTemplate } from "../components/base-template";
import { Button } from "../components/button";
import CardHeader from "../components/card-header";
import { Muted, Paragraph } from "../components/text";
import { colors } from "../components/theme";
import {
  type SharedProps,
  sharedPreviewProps,
  sharedTemplateProps,
} from "../props";

interface EmailVerificationData {
  userFullName: string;
  verificationLink: string;
}

interface EmailVerificationProps extends SharedProps {
  data: EmailVerificationData;
}

export const EmailVerification = ({
  data,
  ...props
}: EmailVerificationProps) => (
  <BaseTemplate
    {...props}
    preview={`Confirm the email address for your ${props.appName} account`}
  >
    <CardHeader title="Verify your email address" />
    <Paragraph>Hello {data.userFullName},</Paragraph>
    <Paragraph>
      Click the button below to confirm the email address for your{" "}
      {props.appName} account. This link expires in 24 hours.
    </Paragraph>

    <Button href={data.verificationLink}>Verify email address</Button>

    <Muted style={{ marginTop: "32px" }}>
      Or if you don't like clicking buttons, open this link:
      <br />
      <Link href={data.verificationLink} style={linkStyle}>
        {data.verificationLink}
      </Link>
    </Muted>
  </BaseTemplate>
);

export default EmailVerification;

const linkStyle = {
  color: colors.mutedForeground,
  textDecoration: "underline",
  wordBreak: "break-all" as const,
};

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
      "https://id.example.com/user/verify-email?code=abcdefg12345",
  },
};
