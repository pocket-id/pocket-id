import { Text } from "@react-email/components";
import { BaseTemplate } from "../components/base-template";
import CardHeader from "../components/card-header";
import { sharedPreviewProps, sharedTemplateProps } from "../props";

interface TestEmailProps {
  logoURL: string;
  appName: string;
}

export const TestEmail = ({ logoURL, appName }: TestEmailProps) => (
  <BaseTemplate logoURL={logoURL} appName={appName}>
    <CardHeader title="Test Email" />
    <Text>Your email setup is working correctly!</Text>
  </BaseTemplate>
);

export default TestEmail;

TestEmail.TemplateProps = {
  ...sharedTemplateProps,
};

TestEmail.PreviewProps = {
  ...sharedPreviewProps,
};
