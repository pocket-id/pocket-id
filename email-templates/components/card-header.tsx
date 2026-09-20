import { Heading } from "react-email";
import { colors, fonts } from "./theme";

interface CardHeaderProps {
  title: string;
}

export default function CardHeader({ title }: CardHeaderProps) {
  return (
    <Heading as="h1" style={titleStyle}>
      {title}
    </Heading>
  );
}

const titleStyle = {
  margin: "0 0 16px 0",
  fontFamily: fonts.serif,
  fontSize: "26px",
  lineHeight: "32px",
  fontWeight: 400,
  letterSpacing: "-0.01em",
  color: colors.foreground,
};
