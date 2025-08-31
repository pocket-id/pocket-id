import { Button as EmailButton } from "@react-email/components";

interface ButtonProps {
  href: string;
  children: React.ReactNode;
  style?: React.CSSProperties;
}

export const Button = ({ href, children, style = {} }: ButtonProps) => {
  const buttonStyle = {
    backgroundColor: "#000000",
    color: "#ffffff",
    padding: "12px 24px",
    borderRadius: "4px",
    fontSize: "15px",
    fontWeight: "500",
    cursor: "pointer",
    marginTop: "10px",
    ...style,
  };

  return (
    <div style={buttonContainer}>
      <EmailButton style={buttonStyle} href={href}>
        {children}
      </EmailButton>
    </div>
  );
};

const buttonContainer = {
  textAlign: "center" as const,
};
