import { Column, Row, Section } from "react-email";
import { colors, radius } from "./theme";

interface DetailsListProps {
  items: { label: string; value: string }[];
}

// A key-value list inside a muted box, used for sign-in details and similar metadata
// Each value sits below its label so long values never fight the label for width on narrow screens
// Every item is its own Row so the plain-text build separates entries with a blank line
export const DetailsList = ({ items }: DetailsListProps) => (
  <Section style={boxStyle}>
    {items.map((item, index) => {
      const isLast = index === items.length - 1;
      return (
        <Row key={item.label}>
          <Column style={isLast ? cellStyle : { ...cellStyle, ...dividerStyle }}>
            <span style={labelStyle}>{item.label}</span>
            <br />
            <span style={valueStyle}>{item.value}</span>
          </Column>
        </Row>
      );
    })}
  </Section>
);

const boxStyle = {
  margin: "8px 0 0 0",
  backgroundColor: colors.muted,
  border: `1px solid ${colors.border}`,
  borderRadius: radius.box,
};

const cellStyle = {
  padding: "12px 16px",
};

const labelStyle = {
  fontSize: "12px",
  lineHeight: "18px",
  color: colors.mutedForeground,
};

const valueStyle = {
  fontSize: "14px",
  lineHeight: "20px",
  fontWeight: 500,
  color: colors.foreground,
};

const dividerStyle = {
  borderBottom: `1px solid ${colors.border}`,
};
