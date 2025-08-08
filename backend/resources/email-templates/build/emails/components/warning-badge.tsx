import { Text } from '@react-email/components';

interface WarningBadgeProps {
  children: React.ReactNode;
}

export const WarningBadge = ({ children }: WarningBadgeProps) => <Text style={warning}>{children}</Text>;

const warning = {
  backgroundColor: '#ffd966',
  color: '#7f6000',
  padding: '4px 12px',
  borderRadius: '50px',
  fontSize: '0.875rem',
  margin: 'auto 0 auto auto',
  display: 'inline-block',
};
