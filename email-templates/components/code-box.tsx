import { Column, Row, Section, Text } from "react-email";
import { colors, fonts, radius } from "./theme";

interface CodeBoxProps {
  code: string;
}

export const CodeBox = ({ code }: CodeBoxProps) => (
  <Section style={boxStyle}>
    <Row>
      <Column style={cellStyle}>
        <Text style={codeStyle}>{code}</Text>
      </Column>
    </Row>
  </Section>
);

const boxStyle = {
  margin: "8px 0 0 0",
  backgroundColor: colors.muted,
  border: `1px solid ${colors.border}`,
  borderRadius: radius.box,
};

const cellStyle = {
  padding: "20px 16px",
  textAlign: "center" as const,
};

const codeStyle = {
  margin: 0,
  fontFamily: fonts.mono,
  fontSize: "32px",
  lineHeight: "40px",
  fontWeight: 600,
  letterSpacing: "8px",
  paddingLeft: "8px",
  color: colors.foreground,
};
