import { Button as EmailButton } from '@react-email/components';

interface ButtonProps {
  href: string;
  children: React.ReactNode;
  style?: React.CSSProperties;
}

export const Button = ({ href, children, style = {} }: ButtonProps) => {
  const buttonStyle = {
    backgroundColor: '#000000',
    color: '#ffffff',
    padding: '0.7rem 1.5rem',
    textDecoration: 'none',
    borderRadius: '4px',
    fontSize: '1rem',
    fontWeight: '500',
    display: 'inline-block',
    border: 'none',
    cursor: 'pointer',
    ...style,
  };

  return (
    <div style={{ textAlign: 'center' as const, marginTop: '24px' }}>
      <EmailButton style={buttonStyle} href={href}>
        {children}
      </EmailButton>
    </div>
  );
};
