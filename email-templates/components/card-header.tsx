import { Column, Heading, Row, Text } from "@react-email/components";

export default function CardHeader({
  title,
  warning,
}: {
  title: string;
  warning?: boolean;
}) {
  return (
    <Row>
      <Column>
        <Heading as="h1" style={titleStyle}>
          {title}
        </Heading>
      </Column>
      <Column align="right">
        {warning && <Text style={warningStyle}>Warning</Text>}
      </Column>
    </Row>
  );
}

const titleStyle = {
  fontSize: "20px",
  fontWeight: "bold" as const,
  margin: 0,
};

const warningStyle = {
  backgroundColor: "#ffd966",
  color: "#7f6000",
  padding: "1px 12px",
  borderRadius: "50px",
  fontSize: "12px",
  display: "inline-block",
};
