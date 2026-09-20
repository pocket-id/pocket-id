import { Button as EmailButton, Section } from "react-email";
import { colors, fonts, radius } from "./theme";

interface ButtonProps {
  href: string;
  children: React.ReactNode;
}

export const Button = ({ href, children }: ButtonProps) => (
  <Section style={containerStyle}>
    <EmailButton href={href} style={buttonStyle}>
      {children}
    </EmailButton>
  </Section>
);

const containerStyle = {
  margin: "24px 0 8px 0",
  textAlign: "center" as const,
};

const buttonStyle = {
  display: "inline-block",
  padding: "12px 28px",
  backgroundColor: colors.primary,
  color: colors.primaryForeground,
  borderRadius: radius.pill,
  fontFamily: fonts.sans,
  fontSize: "14px",
  lineHeight: "20px",
  fontWeight: 500,
  textDecoration: "none",
};
