import { Text } from "react-email";
import { colors } from "./theme";

interface TextProps {
  children: React.ReactNode;
  style?: React.CSSProperties;
}

export const Paragraph = ({ children, style }: TextProps) => (
  <Text style={{ ...paragraphStyle, ...style }}>{children}</Text>
);

export const Muted = ({ children, style }: TextProps) => (
  <Text style={{ ...mutedStyle, ...style }}>{children}</Text>
);

const paragraphStyle = {
  margin: "0 0 16px 0",
  fontSize: "15px",
  lineHeight: "24px",
  color: colors.text,
};

const mutedStyle = {
  margin: "16px 0 0 0",
  fontSize: "13px",
  lineHeight: "20px",
  color: colors.mutedForeground,
};
